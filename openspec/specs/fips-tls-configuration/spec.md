## ADDED Requirements

### Requirement: FIPS-compliant TLS for embedded Tomcat server
The embedded Tomcat HTTPS server SHALL use BCJSSE for all TLS handshakes, using only FIPS-approved cipher suites and protocols.

#### Scenario: Server starts with FIPS TLS
- **WHEN** the scheduler starts with HTTPS enabled on port 8083
- **THEN** the embedded Tomcat server SHALL use an `SSLContext` created by the BCJSSE provider
- **AND** the server SHALL accept TLSv1.2 and TLSv1.3 connections
- **AND** mutual TLS (client-auth: NEED) SHALL continue to function

#### Scenario: Server rejects non-FIPS cipher suites
- **WHEN** a client attempts to connect using a non-FIPS-approved cipher suite
- **THEN** the TLS handshake SHALL fail

#### Scenario: Server accepts connections with FIPS-approved ciphers
- **WHEN** a client connects using `TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384` or `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256`
- **THEN** the TLS handshake SHALL succeed and the request SHALL be processed

### Requirement: FIPS-compliant TLS for scaling engine HTTP client
The outbound HTTP client to the scaling engine SHALL use BCJSSE for TLS, with the `scalingengine` SSL bundle providing certificates.

#### Scenario: Client connects to scaling engine with FIPS TLS
- **WHEN** the scheduler sends a request to the scaling engine via `RestClientConfig`
- **THEN** the `SSLContext` used by Apache HttpClient5 SHALL be created by the BCJSSE provider
- **AND** the client certificate from the `scalingengine` SSL bundle SHALL be presented during mTLS

#### Scenario: Client uses configured TLS protocol
- **WHEN** `client.ssl.protocol` is set to `TLSv1.2`
- **THEN** the `SSLConnectionSocketFactory` SHALL use TLSv1.2 with BCJSSE

### Requirement: FIPS-compliant cipher suite configuration
All TLS configurations SHALL use only FIPS 140-3 approved cipher suites.

#### Scenario: Server cipher suites are FIPS-approved
- **WHEN** the server SSL configuration specifies cipher suites
- **THEN** all configured ciphers SHALL be from the FIPS-approved set (AES-GCM based ciphers with ECDHE or DHE key exchange)

#### Scenario: TLSv1.3 cipher suites
- **WHEN** a TLSv1.3 connection is established
- **THEN** the negotiated cipher suite SHALL be one of `TLS_AES_256_GCM_SHA384` or `TLS_AES_128_GCM_SHA256` (both FIPS-approved)

### Requirement: SSL bundle integration with FIPS providers
Spring Boot SSL bundles SHALL produce `SSLContext` instances backed by BCJSSE when `SslBundle.createSslContext()` is called.

#### Scenario: scalingengine bundle creates FIPS SSLContext
- **WHEN** `sslBundles.getBundle("scalingengine").createSslContext()` is called
- **THEN** the returned `SSLContext` SHALL use the BCJSSE provider
- **AND** the context SHALL contain the keystore and truststore from the PEM bundle configuration

#### Scenario: server bundle creates FIPS SSLContext
- **WHEN** the embedded Tomcat configures SSL using the `server` bundle
- **THEN** the `SSLContext` SHALL use the BCJSSE provider
- **AND** client authentication (NEED) SHALL work with BCFIPS-parsed client certificates
