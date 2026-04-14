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

### 1. Use Bouncy Castle FIPS library family (`bc-fips`, `bctls-fips`, `bcpkix-fips`)

**Rationale**: Bouncy Castle FIPS is the only widely-available, FIPS 140-3 validated cryptographic provider for Java that works as a drop-in JCE/JSSE provider. Alternatives considered:
- **OpenJDK FIPS mode** (via system-level NSS/PKCS#11): Requires OS-level FIPS module configuration, not portable across CF stemcells, and not consistently validated for Java 17+.
- **Corretto Crypto Provider (ACCP)**: Not FIPS 140-3 validated (only FIPS 140-2 for specific versions), and Amazon-specific.
- **SunPKCS11 + Hardware HSM**: Requires hardware, not suitable for CF containers.

**Versions**: Use the latest stable FIPS-validated versions:
- `org.bouncycastle:bc-fips:2.1.2` (core FIPS provider)
- `org.bouncycastle:bctls-fips:2.1.22` (TLS/JSSE provider)
- `org.bouncycastle:bcpkix-fips:2.1.10` (PKI utilities for certificate/key parsing)

### 2. Register providers programmatically via a Spring `@Configuration` class

**Rationale**: Registering in code (via `Security.insertProviderAt()`) is explicit, testable, and does not require modifying `java.security` files in the JRE. A `@Configuration` class with `@Order(Ordered.HIGHEST_PRECEDENCE)` ensures the provider is registered before any Spring SSL bundle initialization. Alternatives considered:
- **`java.security` file modification**: Fragile, requires buildpack customization, hard to test.
- **`-Djava.security.provider` JVM arguments**: Less visible, harder to validate in code.

The configuration class will:
1. Insert `BouncyCastleFipsProvider` at position 1.
2. Insert `BouncyCastleJsseProvider` (with `fips:BCFIPS`) at position 2.
3. Remove the default Sun providers (`SunJSSE`, `SunJCE`, `SunEC`) to ensure FIPS-only operation.
4. Invoke `CryptoServicesRegistrar.setApprovedOnlyMode(true)` to activate FIPS approved-only mode.
5. Validate that `CryptoServicesRegistrar.isInApprovedOnlyMode()` returns `true`; throw a fatal exception otherwise.

### 3. Implement a custom PEM parsing utility using `bcpkix-fips`

**Rationale**: The standard Bouncy Castle (non-FIPS) `PemReader`/`PemParser` classes are **not available** in the FIPS variant. Spring Boot's built-in PEM parsing (`PemContent`) uses the default JCE providers, which will now be BCFIPS — so standard PEM parsing *may* work, but we need a fallback for edge cases where Spring's PEM handling fails under FIPS constraints.

The approach is:
1. **Primary path**: Let Spring Boot's `PemSslStoreHelper` work with BCFIPS as the registered provider. Since Spring Boot 3.1+ uses `CertificateFactory.getInstance("X.509")` and `KeyFactory.getInstance(...)` which delegate to whatever provider is registered, this should work transparently.
2. **Validation**: Add integration tests that confirm Spring Boot PEM bundle loading works end-to-end under BCFIPS.
3. **Fallback utility**: Create a `FipsPemUtils` class that can parse PEM certificates and private keys directly via `bcpkix-fips` APIs (`org.bouncycastle.openssl.jcajce.JcaPEMKeyConverter` from bcpkix-fips, and `CertificateFactory` from BCFIPS) for any cases where direct PEM handling is needed outside of Spring's bundle system (e.g., the `HttpAuthFilter` XFCC parsing).

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

## Risks / Trade-offs

- **[Performance]** BCFIPS may be slower than the default SunJCE for some operations (e.g., AES, RSA). → Mitigation: The scheduler is not crypto-intensive; TLS handshakes happen infrequently. Monitor startup time and response latency after the change.

- **[Spring Boot PEM compatibility]** Spring Boot's internal PEM parsing code may make assumptions about available providers. → Mitigation: Test PEM bundle loading thoroughly under BCFIPS. If Spring's `PemSslStoreHelper` fails, intercept via a custom `SslBundleRegistrar` or pre-convert PEM to in-memory `KeyStore` objects.

- **[Provider ordering conflicts]** Removing Sun providers could break non-TLS crypto (e.g., `SecureRandom`, `MessageDigest` used internally by libraries). → Mitigation: BCFIPS provides implementations for all standard JCA algorithms. Test all functionality end-to-end.

- **[bcpkix-fips PEM key parsing]** EC private keys in PKCS#8 PEM format must be supported for CF Instance Identity keys. → Mitigation: Verify that `bcpkix-fips` `JcaPEMKeyConverter` handles EC keys. CF typically issues EC P-256 keys.

- **[Dependency conflicts]** Bouncy Castle FIPS jars must NOT coexist with non-FIPS Bouncy Castle jars on the classpath. → Mitigation: Verify no transitive dependencies pull in `bcprov-jdk18on` or similar. Use Maven enforcer plugin if needed.

- **[Test infrastructure]** Existing test certificates (self-signed, RSA) must work under BCFIPS. → Mitigation: BCFIPS supports RSA. Regenerate test certs only if they use non-FIPS algorithms (unlikely).