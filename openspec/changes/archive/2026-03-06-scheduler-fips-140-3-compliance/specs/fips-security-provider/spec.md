## ADDED Requirements

### Requirement: FIPS provider registration at startup
The system SHALL register Bouncy Castle FIPS (`BouncyCastleFipsProvider`) as the primary JCE security provider and Bouncy Castle JSSE (`BouncyCastleJsseProvider`) as the primary JSSE provider before any TLS or cryptographic operations occur.

#### Scenario: Successful provider registration
- **WHEN** the scheduler application starts
- **THEN** `BouncyCastleFipsProvider` SHALL be at position 1 in the JVM security provider list
- **AND** `BouncyCastleJsseProvider` (configured with `fips:BCFIPS`) SHALL be at position 2

#### Scenario: Default Sun providers are removed
- **WHEN** the scheduler application has completed provider registration
- **THEN** the `SunJSSE`, `SunJCE`, and `SunEC` providers SHALL NOT be present in the JVM security provider list

### Requirement: FIPS approved-only mode activation
The system SHALL activate Bouncy Castle FIPS approved-only mode at startup, ensuring only FIPS 140-3 validated algorithms are available for cryptographic operations.

#### Scenario: Approved-only mode enabled
- **WHEN** the scheduler application starts and the FIPS provider has been registered
- **THEN** `CryptoServicesRegistrar.isInApprovedOnlyMode()` SHALL return `true`

#### Scenario: Approved-only mode cannot be enabled
- **WHEN** the scheduler application starts and the FIPS provider cannot be set to approved-only mode
- **THEN** the application SHALL throw an `IllegalStateException` and refuse to start
- **AND** the error message SHALL clearly indicate that FIPS approved-only mode could not be activated

### Requirement: Startup validation of FIPS mode
The system SHALL perform a validation check at startup to confirm the FIPS provider is correctly installed and operational, logging the results.

#### Scenario: Successful validation
- **WHEN** the FIPS provider is registered and approved-only mode is active
- **THEN** the system SHALL log all registered security providers at INFO level
- **AND** the system SHALL log a confirmation message that FIPS approved-only mode is active

#### Scenario: Validation failure causes startup abort
- **WHEN** any FIPS validation check fails (provider not registered, approved-only mode not active, or self-test failure)
- **THEN** the scheduler SHALL fail to start with a descriptive error message
- **AND** the error SHALL be logged at ERROR level before the application exits

### Requirement: Provider registration before Spring SSL initialization
The FIPS provider registration SHALL occur before Spring Boot initializes SSL bundles, ensuring that all SSL bundle creation uses BCFIPS and BCJSSE providers.

#### Scenario: Provider available for SSL bundle creation
- **WHEN** Spring Boot processes PEM SSL bundles defined in `application.yml`
- **THEN** the `CertificateFactory` and `KeyFactory` instances used internally SHALL be provided by BCFIPS
