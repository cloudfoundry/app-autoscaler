## 1. Maven Dependencies

- [x] 1.1 Add `org.bouncycastle:bc-fips:2.1.2` dependency to `scheduler/pom.xml`
- [x] 1.2 Add `org.bouncycastle:bctls-fips:2.1.22` dependency to `scheduler/pom.xml`
- [x] 1.3 Add `org.bouncycastle:bcpkix-fips:2.1.10` dependency to `scheduler/pom.xml`
- [x] 1.4 Verify no transitive dependency pulls in non-FIPS Bouncy Castle jars (e.g., `bcprov-jdk18on`); add Maven enforcer exclusion if needed

## 2. FIPS Security Provider Bootstrap

- [x] 2.1 Create `FipsSecurityProviderConfig.java` configuration class in `scheduler/conf/` that registers `BouncyCastleFipsProvider` at position 1 and `BouncyCastleJsseProvider` (with `fips:BCFIPS`) at position 2
- [x] 2.2 Remove default Sun providers (`SunJSSE`, `SunJCE`, `SunEC`) from the JVM security provider list in the configuration class
- [x] 2.3 Activate FIPS approved-only mode via `CryptoServicesRegistrar.setApprovedOnlyMode(true)`
- [x] 2.4 Add startup validation that checks `CryptoServicesRegistrar.isInApprovedOnlyMode()` returns `true` and throws `IllegalStateException` if not
- [x] 2.5 Log all registered security providers at INFO level during startup
- [x] 2.6 Ensure the FIPS provider registration runs before Spring Boot SSL bundle initialization (use `@Order(Ordered.HIGHEST_PRECEDENCE)` or implement as `EnvironmentPostProcessor`)

## 3. PEM Certificate and Key Parsing Utility

- [x] 3.1 Create `FipsPemUtils.java` utility class in `scheduler/util/` with `parseCertificate(String pem)` method that parses PEM X.509 certificates using BCFIPS `CertificateFactory`
- [x] 3.2 Add `parsePrivateKey(String pem)` method to `FipsPemUtils` that handles both RSA and EC (P-256) private keys in PKCS#8 PEM format using BCFIPS-backed `KeyFactory`
- [x] 3.3 Verify that Spring Boot's internal `PemSslStoreHelper` works with BCFIPS as the primary provider for loading PEM SSL bundles (manual integration test)

## 4. TLS Configuration Updates

- [x] 4.1 Verify that `RestClientConfig.java` `SSLContext` creation via `SslBundle.createSslContext()` produces a BCJSSE-backed context without code changes (the provider registration should make this automatic)
- [x] 4.2 Verify that `HttpAuthFilter.java` `CertificateFactory.getInstance("X.509")` routes to BCFIPS without code changes
- [x] 4.3 Verify that the embedded Tomcat HTTPS server (port 8083) uses BCJSSE for TLS handshakes with the existing `server` SSL bundle
- [x] 4.4 Update `application.yml` if any TLS protocol or cipher suite configuration needs adjustment for BCJSSE compatibility (likely no changes needed)

## 5. Unit Tests

- [x] 5.1 Write unit test for `FipsSecurityProviderConfig` that verifies BCFIPS is registered at position 1 and `isInApprovedOnlyMode()` returns `true`
- [x] 5.2 Write unit test for `FipsSecurityProviderConfig` that verifies Sun providers are removed
- [x] 5.3 Write unit tests for `FipsPemUtils.parseCertificate()` with valid PEM, invalid PEM, and empty input
- [x] 5.4 Write unit tests for `FipsPemUtils.parsePrivateKey()` with RSA PKCS#8 PEM and EC P-256 PKCS#8 PEM
- [x] 5.5 Write integration test that loads the Spring application context with BCFIPS and verifies both SSL bundles (`server` and `scalingengine`) are created successfully

## 6. Existing Test Compatibility

- [x] 6.1 Run full scheduler test suite (`make scheduler.test`) and fix any failures caused by the FIPS provider switch
- [x] 6.2 Verify `EmbeddedTomcatUtil` test utility works with BCFIPS (PKCS12 keystore loading and TLS handshake)
- [x] 6.3 Verify `HttpAuthFilter` tests pass with BCFIPS-backed certificate parsing
- [x] 6.4 Verify `RestClientConfig` and scaling engine client tests pass with BCJSSE SSL context
