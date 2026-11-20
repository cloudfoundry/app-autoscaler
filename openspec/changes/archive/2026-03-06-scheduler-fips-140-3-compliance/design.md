## Context

The Application Autoscaler for Cloud Foundry consists of 7 microservices. Six are written in Go and already support FIPS 140 compliance via `GOFIPS140=v1.0.0`. The seventh — the **Scheduler** — is a Java 17 / Spring Boot 3.5.11 application that currently uses the default JVM security providers (SunJSSE, SunJCE) with no FIPS awareness.

The scheduler handles TLS in three places:

1. **Embedded Tomcat HTTPS server** — accepts mTLS connections from the API server and GoRouter, configured via Spring Boot PEM SSL bundles (`application.yml` lines 53-67).
2. **Outbound HTTP client to Scaling Engine** — `RestClientConfig.java` creates an `SSLContext` from the `scalingengine` SSL bundle using Apache HttpClient5.
3. **XFCC certificate validation** — `HttpAuthFilter.java` parses X.509 certificates from the `X-Forwarded-Client-Cert` header using `CertificateFactory.getInstance("X.509")`.

Additionally, `CloudFoundryConfigurationProcessor.java` overrides SSL bundle paths at startup using CF Instance Identity certificates from environment variables (`CF_INSTANCE_CERT`, `CF_INSTANCE_KEY`, `CF_INSTANCE_CA_CERT`).

Spring Boot 3.5's PEM SSL bundle support internally uses `PemContent` / `PemSslStoreHelper` which depend on the default `CertificateFactory` and `KeyFactory` providers. These must work through a FIPS-certified provider.

## Goals / Non-Goals

**Goals:**
- Register Bouncy Castle FIPS (`BCFIPS`) as the JVM's primary security provider so all cryptographic operations use FIPS 140-3 validated algorithms.
- Register Bouncy Castle JSSE provider (`BCJSSE`) for TLS handshake operations.
- Validate at startup that the FIPS provider is in approved-only mode; fail fast if not.
- Ensure PEM-encoded certificates and private keys (as used by Spring Boot SSL bundles and CF Instance Identity) can be parsed under the FIPS provider.
- Ensure existing TLS connections (server mTLS, client to scaling engine, XFCC parsing) continue to work unchanged.
- Maintain backward compatibility with the existing `application.yml` configuration structure.

**Non-Goals:**
- FIPS compliance for the PostgreSQL JDBC driver's internal TLS (the driver manages its own SSL context; this is out of scope for this change).
- Replacing Apache HttpClient5 with a different HTTP client library.
- Supporting non-PEM certificate formats (PKCS12, JKS) under FIPS — the project uses PEM exclusively.
- Certifying the application itself under FIPS — only the cryptographic provider needs to be FIPS-validated.
- Changes to Go microservices (already FIPS-compliant).

## Decisions

### 1. Use Bouncy Castle FIPS providers (`bc-fips`, `bctls-fips`)

**Rationale**: Bouncy Castle FIPS is the only widely-available, FIPS 140-3 validated cryptographic provider for Java that works as a drop-in JCE/JSSE provider. Alternatives considered:
- **OpenJDK FIPS mode** (via system-level NSS/PKCS#11): Requires OS-level FIPS module configuration, not portable across CF stemcells, and not consistently validated for Java 17+.
- **Corretto Crypto Provider (ACCP)**: Not FIPS 140-3 validated (only FIPS 140-2 for specific versions), and Amazon-specific.
- **SunPKCS11 + Hardware HSM**: Requires hardware, not suitable for CF containers.

**Versions**: Use the latest stable FIPS-validated versions:
- `org.bouncycastle:bc-fips:2.1.2` (core FIPS provider)
- `org.bouncycastle:bctls-fips:2.1.22` (TLS/JSSE provider)

### 2. Register providers in an early Spring `EnvironmentPostProcessor`

**Rationale**: Registering in code (via `Security.insertProviderAt()`) is explicit and testable. `FipsEnvironmentPostProcessor` runs after Cloud Foundry configuration mapping and before application context creation. This initializes application-managed cryptography for both executable startup and the Cloud Foundry Java buildpack's external-Tomcat WAR startup, which bypasses `SchedulerApplication.main()`. Container-managed TLS remains the runtime's responsibility. Alternatives considered:
- **`java.security` file modification**: Fragile, requires buildpack customization, hard to test.
- **`-Djava.security.provider` JVM arguments**: Less visible, harder to validate in code.

Boot's `CloudFoundryVcapEnvironmentPostProcessor` parses `VCAP_SERVICES` and `VCAP_APPLICATION` into `vcap.services.*` and `vcap.application.*`. The scheduler uses `Binder` to select services by tag and map their credentials to scheduler properties; it does not parse VCAP JSON itself.

When `autoscaler.fips_mode` is true, the FIPS processor will:
1. Set `org.bouncycastle.fips.approved_only=true` before Bouncy Castle initializes, establishing approved-only mode for all threads.
2. Insert `BouncyCastleFipsProvider` at position 1.
3. Insert `BouncyCastleJsseProvider` (with `fips:BCFIPS`) at position 2.
4. Remove `SunJSSE` and `SunEC`, and set the default key/trust manager algorithms to `PKIX`.
5. Validate approved-only mode and propagate activation failures so startup aborts.

Fresh-JVM tests exercise WAR bootstrap using the VCAP flag without manually initializing FIPS. They verify provider registration before bean creation, BCJSSE SSL bundle contexts, approved-only mode on a worker thread, disabled/absent flags, and startup failure when BC was initialized too early in unapproved mode.

### 3. Reuse Spring Boot PEM parsing and SSL bundles

**Rationale**: Spring Boot's public `PemContent` API supports RSA PKCS#1, EC SEC1, PKCS#8 private keys, and certificate chains. Its standard JCA factory calls use BCFIPS when registered as the primary provider. Compatibility tests cover these formats in approved-only mode, so a second parser and `bcpkix-fips` dependency are unnecessary.

The approach is:
1. **Client identity**: Configure the full `CF_INSTANCE_CERT` chain and `CF_INSTANCE_KEY` through the existing PEM SSL bundle. `CF_INSTANCE_CA_CERT` is optional and is not required to load the identity. Spring Boot creates the key managers and propagates errors loading a configured identity.
2. **Standalone PEM parsing**: Use `PemContent.load(path).getCertificates()` to load CF trust anchors. Keep system/custom trust merging separate from PEM parsing.
3. **Validation**: Test RSA/EC keys, certificate chain order, bundle key managers, and BCJSSE SSL context creation under BCFIPS, plus PEM parsing with default JVM providers.

### 4. Keep `HttpAuthFilter` certificate parsing using standard `CertificateFactory` API

**Rationale**: The `HttpAuthFilter.parseCertificate()` method uses `CertificateFactory.getInstance("X.509")`, which is a standard JCA API. Once BCFIPS is registered as the primary provider, this call will automatically route to BCFIPS's `CertificateFactory` implementation. No code changes needed in the filter itself — only provider registration.

### 5. Configure TLS protocol to TLSv1.2 and TLSv1.3 with FIPS-approved cipher suites

**Rationale**: BCJSSE supports both TLSv1.2 and TLSv1.3. The existing cipher suite list (`TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384`, `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256`) is FIPS-approved. TLSv1.3 cipher suites (`TLS_AES_256_GCM_SHA384`, `TLS_AES_128_GCM_SHA256`) are also FIPS-approved. No cipher suite changes are required.

### 6. Startup validation pattern: fail-fast with descriptive logging

**Rationale**: If FIPS mode cannot be confirmed, the scheduler must refuse to start. This prevents silently running in a non-compliant mode. The validation will:
- Log all registered security providers at startup.
- Confirm BCFIPS approved-only mode.
- Perform a self-test (e.g., create an `SSLContext` with BCJSSE to verify TLS works).
- Throw `IllegalStateException` with a clear message if any check fails.

### 7. Compose outbound HTTPS trust anchors and let Spring Boot create the SSL context

`CfTrustStore` combines the JVM's default trust anchors, CF/BOSH CA certificates, and the configured SSL bundle truststore into one certificate-only `KeyStore`, deduplicating certificates without modifying the input store. Default anchors come from `TrustManagerFactory.init((KeyStore) null)`, which honors `javax.net.ssl.trustStore*` and the JSSE default-store search rather than hardcoding `cacerts` or its password.

CF certificates are read from `CF_CA_CERTS`, defaulting to `/etc/ssl/certs/ca-certificates.crt`. The default file may be absent outside CF; an explicit unreadable path or malformed certificates fails initialization. Explicit loading is needed because the buildpack's container trust-manager factory delegates to `SunJSSE`, which is removed in FIPS mode. BCJSSE itself supports JVM default trust.

`RestClientConfig` combines this truststore with the original bundle's key store, password, key alias, and options, then calls `SslBundle.createSslContext()` using the configured client protocol. This uses the provider's own trust manager without a custom validation wrapper. The client's identity chain stays in the key store; it is not added to outbound trust anchors.

Local HTTPS tests verify mutual TLS using each trust source under TLS 1.2 and TLS 1.3, full client-chain presentation, rejection of untrusted server certificates, and hostname verification.

## Risks / Trade-offs

- **[Performance]** BCFIPS may be slower than the default SunJCE for some operations (e.g., AES, RSA). → Mitigation: The scheduler is not crypto-intensive; TLS handshakes happen infrequently. Monitor startup time and response latency after the change.

- **[Spring Boot PEM compatibility]** Spring Boot's internal PEM parsing code may make assumptions about available providers. → Mitigation: Test PEM bundle loading thoroughly under BCFIPS. If Spring's `PemSslStoreHelper` fails, intercept via a custom `SslBundleRegistrar` or pre-convert PEM to in-memory `KeyStore` objects.

- **[Provider ordering conflicts]** Removing Sun providers could break non-TLS crypto (e.g., `SecureRandom`, `MessageDigest` used internally by libraries). → Mitigation: BCFIPS provides implementations for all standard JCA algorithms. Test all functionality end-to-end.

- **[PEM key parsing]** EC private keys must be supported for CF Instance Identity keys. → Mitigation: Verify Spring Boot PEM parsing under BCFIPS with EC P-256 keys, alongside RSA PKCS#1 and PKCS#8 fixtures.

- **[Dependency conflicts]** Bouncy Castle FIPS jars must NOT coexist with non-FIPS Bouncy Castle jars on the classpath. → Mitigation: Verify no transitive dependencies pull in `bcprov-jdk18on` or similar. Use Maven enforcer plugin if needed.

- **[Test infrastructure]** Existing test certificates (self-signed, RSA) must work under BCFIPS. → Mitigation: BCFIPS supports RSA. Regenerate test certs only if they use non-FIPS algorithms (unlikely).
