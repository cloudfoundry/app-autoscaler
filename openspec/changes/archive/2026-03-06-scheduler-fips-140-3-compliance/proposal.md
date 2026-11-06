## Why

The scheduler component (Java/Spring Boot) must run in FIPS 140-3 compliant mode to meet regulatory and security requirements. The Go microservices already support FIPS via `GOFIPS140=v1.0.0`, but the scheduler — the only Java component — has no FIPS provider configured. Cloud Foundry deployments require parsing PEM-encoded Instance Identity certificates at runtime, which is not natively supported by Bouncy Castle FIPS and must be handled explicitly.

## What Changes

- **BREAKING**: Replace the default Java security providers with Bouncy Castle FIPS (bcfips) as the sole cryptographic provider for TLS, certificate parsing, and key management.
- Add Bouncy Castle FIPS libraries (`bc-fips`, `bctls-fips`, `bcpkix-fips`) as Maven dependencies.
- Implement startup validation that confirms the JVM is running in FIPS-approved mode only.
- Reimplement PEM certificate/key parsing to work with Bouncy Castle FIPS, since the standard `PemReader` from non-FIPS Bouncy Castle is unavailable.
- Update Spring Boot SSL bundle configuration to use a FIPS-compliant `SSLContext` and `KeyManagerFactory`/`TrustManagerFactory`.
- Update the `HttpAuthFilter` XFCC certificate parsing to use FIPS-compliant `CertificateFactory`.
- Update the `RestClientConfig` HTTP client SSL setup to use FIPS-compliant providers.
- Update the `CloudFoundryConfigurationProcessor` database SSL certificate handling to be FIPS-compliant.

## Capabilities

### New Capabilities
- `fips-security-provider`: Bootstrap and validate Bouncy Castle FIPS as the JVM's security provider, ensuring FIPS-approved-mode-only operation at startup.
- `fips-pem-certificate-handling`: PEM certificate and private key parsing using only FIPS-approved APIs, replacing the default Java and Spring Boot PEM readers for Cloud Foundry Instance Identity certificates.
- `fips-tls-configuration`: FIPS-compliant TLS context creation for both the embedded server (Tomcat) and outbound HTTP clients (scaling engine), integrating with Spring Boot's SSL bundle abstraction.

### Modified Capabilities
<!-- No existing specs to modify -->

## Impact

- **Dependencies**: New Maven dependencies on `org.bouncycastle:bc-fips`, `org.bouncycastle:bctls-fips`, `org.bouncycastle:bcpkix-fips`.
- **Code**: Changes to `SchedulerApplication.java` (provider registration + validation), `HttpAuthFilter.java` (XFCC cert parsing), `RestClientConfig.java` (SSL context), `CloudFoundryConfigurationProcessor.java` (DB cert handling), new PEM utility class, new FIPS bootstrap configuration class.
- **Configuration**: `application.yml` SSL bundle and server SSL settings may need provider hints.
- **Testing**: All existing TLS-related tests must pass under the FIPS provider. New unit tests for FIPS validation and PEM parsing.
- **Runtime**: The scheduler will refuse to start if FIPS-approved mode cannot be confirmed, acting as a fail-safe.