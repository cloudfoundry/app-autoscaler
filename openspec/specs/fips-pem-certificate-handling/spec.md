## ADDED Requirements

### Requirement: PEM certificate parsing under FIPS provider
The system SHALL be able to parse X.509 certificates from PEM-encoded input using the BCFIPS `CertificateFactory` implementation, without relying on non-FIPS providers.

#### Scenario: Parse PEM certificate from XFCC header
- **WHEN** the `HttpAuthFilter` receives an `X-Forwarded-Client-Cert` header containing a PEM-encoded X.509 certificate
- **THEN** the system SHALL successfully parse the certificate using `CertificateFactory.getInstance("X.509")` backed by BCFIPS
- **AND** the system SHALL extract the subject's X.500 principal for organizational unit validation

#### Scenario: Parse PEM certificate from CF Instance Identity
- **WHEN** the `CloudFoundryConfigurationProcessor` reads `CF_INSTANCE_CERT` containing a PEM-encoded certificate
- **THEN** the system SHALL successfully configure the SSL bundle keystore with the parsed certificate

#### Scenario: Invalid PEM input
- **WHEN** a malformed PEM certificate is provided
- **THEN** the system SHALL throw a `CertificateException` with a descriptive message

### Requirement: PEM private key parsing under FIPS provider
The system SHALL be able to parse private keys from PEM-encoded input (PKCS#8 format) using FIPS-approved APIs.

#### Scenario: Parse RSA private key from PEM
- **WHEN** a PEM-encoded RSA private key (PKCS#8 format) is provided via SSL bundle configuration
- **THEN** the system SHALL successfully create a `PrivateKey` object using BCFIPS-backed `KeyFactory`

#### Scenario: Parse EC private key from PEM
- **WHEN** a PEM-encoded EC private key (PKCS#8 format, P-256 curve) is provided via CF Instance Identity (`CF_INSTANCE_KEY`)
- **THEN** the system SHALL successfully create a `PrivateKey` object using BCFIPS-backed `KeyFactory`

### Requirement: PEM utility for direct parsing outside Spring bundles
The system SHALL provide a utility class (`FipsPemUtils`) for parsing PEM certificates and private keys directly, for use in components that do not go through Spring's SSL bundle system.

#### Scenario: Parse certificate from raw PEM string
- **WHEN** `FipsPemUtils.parseCertificate(String pem)` is called with a valid PEM-encoded certificate
- **THEN** it SHALL return an `X509Certificate` object parsed via BCFIPS `CertificateFactory`

#### Scenario: Parse private key from raw PEM string
- **WHEN** `FipsPemUtils.parsePrivateKey(String pem)` is called with a valid PKCS#8 PEM-encoded private key
- **THEN** it SHALL return a `PrivateKey` object created via BCFIPS-backed key factories

#### Scenario: Spring Boot PEM SSL bundle loads under FIPS
- **WHEN** Spring Boot initializes PEM SSL bundles (`spring.ssl.bundle.pem.*`) with BCFIPS as the primary provider
- **THEN** the `server` and `scalingengine` SSL bundles SHALL be created successfully
- **AND** both keystore (certificate + private key) and truststore (CA certificate) SHALL be populated correctly
