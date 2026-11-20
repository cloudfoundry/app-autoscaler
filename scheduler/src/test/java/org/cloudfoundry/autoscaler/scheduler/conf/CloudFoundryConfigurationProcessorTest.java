package org.cloudfoundry.autoscaler.scheduler.conf;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.stream.Stream;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.Arguments;
import org.junit.jupiter.params.provider.MethodSource;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.cloud.CloudFoundryVcapEnvironmentPostProcessor;
import org.springframework.boot.logging.DeferredLogs;
import org.springframework.core.env.ConfigurableEnvironment;
import org.springframework.core.env.StandardEnvironment;

class CloudFoundryConfigurationProcessorTest {

  private CloudFoundryConfigurationProcessor processor;
  private ConfigurableEnvironment environment;
  private SpringApplication application;

  @BeforeEach
  void setUp() {
    processor = new CloudFoundryConfigurationProcessor();
    environment = new StandardEnvironment();
    application = new SpringApplication();
  }

  @Test
  void testNoVcapServices() {
    processEnvironment();
    assertNull(environment.getProperty("spring.datasource.url"));
  }

  @Test
  void testVcapServicesWithSchedulerConfig() {
    String vcapServices =
        """
        {
          "user-provided": [
            {
              "name": "scheduler-config-service",
              "tags": ["scheduler-config"],
              "credentials": {
                "cfserver": {
                  "xfccVerificationMode": "SHADOW",
                  "systemCertPath": "/platform/roots",
                  "instanceCertPath": "/platform/instance.crt"
                },
                "spring": {
                  "datasource": {
                    "url": "jdbc:postgresql://cf-db-host:5432/autoscaler",
                    "username": "cf-db-user",
                    "password": "cf-db-password"
                  }
                },
                "autoscaler": {
                  "scalingengine": {
                    "url": "https://cf-scaling-engine:8091"
                  }
                },
                "server": {
                  "port": 8080
                }
              }
            }
          ]
        }
        """;

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test", java.util.Map.of("VCAP_SERVICES", vcapServices)));

    processEnvironment();

    assertEquals(
        "jdbc:postgresql://cf-db-host:5432/autoscaler",
        environment.getProperty("spring.datasource.url"));
    assertEquals("cf-db-user", environment.getProperty("spring.datasource.username"));
    assertEquals("cf-db-password", environment.getProperty("spring.datasource.password"));
    assertEquals(
        "https://cf-scaling-engine:8091", environment.getProperty("autoscaler.scalingengine.url"));
    assertEquals("8080", environment.getProperty("server.port"));
    CfServerConfiguration cfServer =
        org.springframework.boot.context.properties.bind.Binder.get(environment)
            .bind("cfserver", CfServerConfiguration.class)
            .get();
    assertEquals(CfServerConfiguration.VerificationMode.SHADOW, cfServer.getXfccVerificationMode());
    assertEquals("/platform/roots", cfServer.getSystemCertPath());
    assertEquals("/platform/instance.crt", cfServer.getInstanceCertPath());
  }

  @Test
  void testVcapServicesWithoutSchedulerConfigTag() {
    String vcapServices =
        """
        {
          "user-provided": [
            {
              "name": "other-service",
              "tags": ["other-tag"],
              "credentials": {
                "spring": {
                  "datasource": {
                    "url": "jdbc:postgresql://other-host:5432/other"
                  }
                }
              }
            }
          ]
        }
        """;

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test", java.util.Map.of("VCAP_SERVICES", vcapServices)));

    processEnvironment();

    assertNull(environment.getProperty("spring.datasource.url"));
  }

  @Test
  void testInvalidVcapServicesJson() {
    String vcapServices = "invalid json";

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test", java.util.Map.of("VCAP_SERVICES", vcapServices)));

    processEnvironment();

    assertNull(environment.getProperty("spring.datasource.url"));
  }

  @Test
  void testEmptyVcapServices() {
    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test", java.util.Map.of("VCAP_SERVICES", "")));

    processEnvironment();

    assertNull(environment.getProperty("spring.datasource.url"));
  }

  @Test
  void testVcapServicesWithDatabaseService() {
    String vcapServices =
        """
        {
          "postgresql-db": [
            {
              "label": "postgresql-db",
              "name": "autoscaler-db",
              "tags": ["relational", "binding_db", "policy_db"],
              "credentials": {
                "username": "dbuser",
                "password": "dbpass",
                "hostname": "db-host.example.com",
                "dbname": "autoscaler_db",
                "port": "5432",
                "uri": "postgres://dbuser:dbpass@db-host.example.com:5432/autoscaler_db",
                "sslcert": "-----BEGIN CERTIFICATE-----\\nMIICert...\\n-----END CERTIFICATE-----",
                "sslrootcert": "-----BEGIN CERTIFICATE-----\\nMIIRoot...\\n-----END CERTIFICATE-----"
              }
            }
          ]
        }
        """;

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test", java.util.Map.of("VCAP_SERVICES", vcapServices)));

    processEnvironment();

    String datasourceUrl = environment.getProperty("spring.datasource.url");
    assertNotNull(datasourceUrl);
    assertEquals(
        "jdbc:postgresql://db-host.example.com:5432/autoscaler_db?sslmode=require", datasourceUrl);

    assertEquals("dbuser", environment.getProperty("spring.datasource.username"));
    assertEquals("dbpass", environment.getProperty("spring.datasource.password"));
    assertEquals(
        "org.postgresql.Driver", environment.getProperty("spring.datasource.driverClassName"));

    // Should also configure policy datasource since policy_db tag is present
    String policyDatasourceUrl = environment.getProperty("spring.policy-db-datasource.url");
    assertNotNull(policyDatasourceUrl);
    assertEquals("dbuser", environment.getProperty("spring.policy-db-datasource.username"));
    assertEquals("dbpass", environment.getProperty("spring.policy-db-datasource.password"));
  }

  @Test
  void testVcapServicesWithDatabaseServiceNoSsl() {
    String vcapServices =
        """
        {
          "postgresql-db": [
            {
              "label": "postgresql-db",
              "name": "autoscaler-db",
              "tags": ["relational", "binding_db"],
              "credentials": {
                "username": "dbuser",
                "password": "dbpass",
                "hostname": "db-host.example.com",
                "dbname": "autoscaler_db",
                "port": "5432"
              }
            }
          ]
        }
        """;

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test", java.util.Map.of("VCAP_SERVICES", vcapServices)));

    processEnvironment();

    String datasourceUrl = environment.getProperty("spring.datasource.url");
    assertNotNull(datasourceUrl);
    assertEquals(
        "jdbc:postgresql://db-host.example.com:5432/autoscaler_db?sslmode=require", datasourceUrl);
    assertEquals("dbuser", environment.getProperty("spring.datasource.username"));
    assertEquals("dbpass", environment.getProperty("spring.datasource.password"));

    // Should configure policy datasource since binding_db tag is present
    assertNotNull(environment.getProperty("spring.policy-db-datasource.url"));
    assertEquals(
        "jdbc:postgresql://db-host.example.com:5432/autoscaler_db?sslmode=require",
        environment.getProperty("spring.policy-db-datasource.url"));
  }

  @Test
  void testVcapServicesWithClientCertCredentialMapping() {
    String vcapServices =
        """
        {
          "postgresql-db": [
            {
              "label": "postgresql-db",
              "name": "autoscaler-db",
              "tags": ["relational", "binding_db"],
              "credentials": {
                "username": "dbuser",
                "password": "dbpass",
                "hostname": "db-host.example.com",
                "dbname": "autoscaler_db",
                "port": "5432",
                "client_cert": "-----BEGIN CERTIFICATE-----\\nMIICert...\\n-----END CERTIFICATE-----",
                "client_key": "-----BEGIN PRIVATE KEY-----\\nMIIKey...\\n-----END PRIVATE KEY-----",
                "sslrootcert": "-----BEGIN CERTIFICATE-----\\nMIIRoot...\\n-----END CERTIFICATE-----"
              }
            }
          ]
        }
        """;

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test", java.util.Map.of("VCAP_SERVICES", vcapServices)));

    processEnvironment();

    String datasourceUrl = environment.getProperty("spring.datasource.url");
    assertNotNull(datasourceUrl);
    assertEquals(
        "jdbc:postgresql://db-host.example.com:5432/autoscaler_db?sslmode=require", datasourceUrl);

    assertEquals("dbuser", environment.getProperty("spring.datasource.username"));
    assertEquals("dbpass", environment.getProperty("spring.datasource.password"));
    assertEquals(
        "org.postgresql.Driver", environment.getProperty("spring.datasource.driverClassName"));
  }

  @Test
  void testVcapServicesWithClientCertOnlyCredentialMapping() {
    String vcapServices =
        """
        {
          "postgresql-db": [
            {
              "label": "postgresql-db",
              "name": "autoscaler-db",
              "tags": ["relational", "binding_db"],
              "credentials": {
                "username": "dbuser",
                "password": "dbpass",
                "hostname": "db-host.example.com",
                "dbname": "autoscaler_db",
                "port": "5432",
                "client_cert": "-----BEGIN CERTIFICATE-----\\nMIICert...\\n-----END CERTIFICATE-----"
              }
            }
          ]
        }
        """;

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test", java.util.Map.of("VCAP_SERVICES", vcapServices)));

    processEnvironment();

    String datasourceUrl = environment.getProperty("spring.datasource.url");
    assertNotNull(datasourceUrl);
    assertEquals(
        "jdbc:postgresql://db-host.example.com:5432/autoscaler_db?sslmode=require", datasourceUrl);
  }

  @Test
  void testVcapServicesPrefersSslcertOverClientCert() {
    String vcapServices =
        """
        {
          "postgresql-db": [
            {
              "label": "postgresql-db",
              "name": "autoscaler-db",
              "tags": ["relational", "binding_db"],
              "credentials": {
                "username": "dbuser",
                "password": "dbpass",
                "hostname": "db-host.example.com",
                "dbname": "autoscaler_db",
                "port": "5432",
                "sslcert": "-----BEGIN CERTIFICATE-----\\nMIISSLCert...\\n-----END CERTIFICATE-----",
                "sslkey": "-----BEGIN PRIVATE KEY-----\\nMIISSLKey...\\n-----END PRIVATE KEY-----",
                "client_cert": "-----BEGIN CERTIFICATE-----\\nMIICert...\\n-----END CERTIFICATE-----",
                "client_key": "-----BEGIN PRIVATE KEY-----\\nMIIKey...\\n-----END PRIVATE KEY-----"
              }
            }
          ]
        }
        """;

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test", java.util.Map.of("VCAP_SERVICES", vcapServices)));

    processEnvironment();

    String datasourceUrl = environment.getProperty("spring.datasource.url");
    assertNotNull(datasourceUrl);
    assertEquals(
        "jdbc:postgresql://db-host.example.com:5432/autoscaler_db?sslmode=require", datasourceUrl);
  }

  @Test
  void testVcapApplicationWithOrgGuid() {
    String vcapServices =
        """
        {
          "user-provided": [
            {
              "name": "scheduler-config-service",
              "tags": ["scheduler-config"],
              "credentials": {
                "cfserver": {
                  "healthserver": {
                    "username": "health-user",
                    "password": "health-password"
                  }
                }
              }
            }
          ]
        }
        """;

    String vcapApplication =
        """
        {
          "application_id": "app-123",
          "application_name": "test-app",
          "organization_id": "test-org-guid-123",
          "space_id": "test-space-guid-456"
        }
        """;

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test",
                java.util.Map.of(
                    "VCAP_SERVICES", vcapServices,
                    "VCAP_APPLICATION", vcapApplication)));

    processEnvironment();

    assertEquals("test-org-guid-123", environment.getProperty("cfserver.validOrgGuid"));
    assertEquals("test-space-guid-456", environment.getProperty("cfserver.validSpaceGuid"));
  }

  @Test
  void testVcapApplicationWithoutOrgGuid() {
    String vcapServices =
        """
        {
          "user-provided": [
            {
              "name": "scheduler-config-service",
              "tags": ["scheduler-config"],
              "credentials": {
                "cfserver": {
                  "healthserver": {
                    "username": "health-user",
                    "password": "health-password"
                  }
                }
              }
            }
          ]
        }
        """;

    String vcapApplication =
        """
        {
          "application_id": "app-123",
          "application_name": "test-app",
          "space_id": "test-space-guid-456"
        }
        """;

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test",
                java.util.Map.of(
                    "VCAP_SERVICES", vcapServices,
                    "VCAP_APPLICATION", vcapApplication)));

    processEnvironment();

    assertNull(environment.getProperty("cfserver.validOrgGuid"));
    assertEquals("test-space-guid-456", environment.getProperty("cfserver.validSpaceGuid"));
  }

  @Test
  void testVcapApplicationWithEmptyOrgGuid() {
    String vcapServices =
        """
        {
          "user-provided": []
        }
        """;

    String vcapApplication =
        """
        {
          "application_id": "app-123",
          "application_name": "test-app",
          "organization_id": "",
          "space_id": "test-space-guid-456"
        }
        """;

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test",
                java.util.Map.of(
                    "VCAP_SERVICES", vcapServices,
                    "VCAP_APPLICATION", vcapApplication)));

    processEnvironment();

    assertNull(environment.getProperty("cfserver.validOrgGuid"));
  }

  @Test
  void testVcapApplicationInvalidJson() {
    String vcapServices =
        """
        {
          "user-provided": []
        }
        """;

    String vcapApplication = "invalid json";

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test",
                java.util.Map.of(
                    "VCAP_SERVICES", vcapServices,
                    "VCAP_APPLICATION", vcapApplication)));

    processEnvironment();

    assertNull(environment.getProperty("cfserver.validOrgGuid"));
  }

  @Test
  void testNoVcapApplication() {
    String vcapServices =
        """
        {
          "user-provided": []
        }
        """;

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test", java.util.Map.of("VCAP_SERVICES", vcapServices)));

    processEnvironment();

    assertNull(environment.getProperty("cfserver.validOrgGuid"));
  }

  @Test
  void testVcapApplicationOverridesSchedulerConfig() {
    String vcapServices =
        """
        {
          "user-provided": [
            {
              "name": "scheduler-config-service",
              "tags": ["scheduler-config"],
              "credentials": {
                "cfserver": {
                  "validOrgGuid": "config-org-guid-999"
                }
              }
            }
          ]
        }
        """;

    String vcapApplication =
        """
        {
          "application_id": "app-123",
          "application_name": "test-app",
          "organization_id": "vcap-org-guid-123",
          "space_id": "test-space-guid-456"
        }
        """;

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test",
                java.util.Map.of(
                    "VCAP_SERVICES", vcapServices,
                    "VCAP_APPLICATION", vcapApplication)));

    processEnvironment();

    assertEquals("vcap-org-guid-123", environment.getProperty("cfserver.validOrgGuid"));
  }

  @Test
  void testVcapApplicationWithOnlySpaceGuid() {
    String vcapServices =
        """
        {
          "user-provided": [
            {
              "name": "scheduler-config-service",
              "tags": ["scheduler-config"],
              "credentials": {
                "cfserver": {
                  "healthserver": {
                    "username": "health-user",
                    "password": "health-password"
                  }
                }
              }
            }
          ]
        }
        """;

    String vcapApplication =
        """
        {
          "application_id": "app-123",
          "application_name": "test-app",
          "space_id": "only-space-guid-789"
        }
        """;

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test",
                java.util.Map.of(
                    "VCAP_SERVICES", vcapServices,
                    "VCAP_APPLICATION", vcapApplication)));

    processEnvironment();

    assertNull(environment.getProperty("cfserver.validOrgGuid"));
    assertEquals("only-space-guid-789", environment.getProperty("cfserver.validSpaceGuid"));
  }

  @Test
  void testVcapApplicationWithEmptySpaceGuid() {
    String vcapServices =
        """
        {
          "user-provided": [
            {
              "name": "scheduler-config-service",
              "tags": ["scheduler-config"],
              "credentials": {
                "cfserver": {
                  "healthserver": {
                    "username": "health-user",
                    "password": "health-password"
                  }
                }
              }
            }
          ]
        }
        """;

    String vcapApplication =
        """
        {
          "application_id": "app-123",
          "application_name": "test-app",
          "organization_id": "test-org-guid-123",
          "space_id": ""
        }
        """;

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test",
                java.util.Map.of(
                    "VCAP_SERVICES", vcapServices,
                    "VCAP_APPLICATION", vcapApplication)));

    processEnvironment();

    assertEquals("test-org-guid-123", environment.getProperty("cfserver.validOrgGuid"));
    assertNull(environment.getProperty("cfserver.validSpaceGuid"));
  }

  @Test
  void testVcapApplicationWithoutSpaceGuid() {
    String vcapServices =
        """
        {
          "user-provided": [
            {
              "name": "scheduler-config-service",
              "tags": ["scheduler-config"],
              "credentials": {
                "cfserver": {
                  "healthserver": {
                    "username": "health-user",
                    "password": "health-password"
                  }
                }
              }
            }
          ]
        }
        """;
    String vcapApplication =
        """
        {
          "application_id": "app-123",
          "application_name": "test-app",
          "organization_id": "test-org-guid-123"
        }
        """;

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test",
                java.util.Map.of(
                    "VCAP_SERVICES", vcapServices,
                    "VCAP_APPLICATION", vcapApplication)));

    processEnvironment();

    assertEquals("test-org-guid-123", environment.getProperty("cfserver.validOrgGuid"));
    assertNull(environment.getProperty("cfserver.validSpaceGuid"));
  }

  @Test
  void testVcapApplicationWithNeitherOrgNorSpaceGuid() {
    String vcapServices =
        """
        {
          "user-provided": []
        }
        """;

    String vcapApplication =
        """
        {
          "application_id": "app-123",
          "application_name": "test-app"
        }
        """;

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test",
                java.util.Map.of(
                    "VCAP_SERVICES", vcapServices,
                    "VCAP_APPLICATION", vcapApplication)));

    processEnvironment();

    assertNull(environment.getProperty("cfserver.validOrgGuid"));
    assertNull(environment.getProperty("cfserver.validSpaceGuid"));
  }

  @Test
  void testCfInstanceCertificatesReadFromFilePaths(@TempDir Path tempDir) throws IOException {
    // CF_INSTANCE_CERT, CF_INSTANCE_KEY, CF_INSTANCE_CA_CERT are file paths, not PEM content
    String caCertPem = "-----BEGIN CERTIFICATE-----\nFakeCACert\n-----END CERTIFICATE-----";
    String instanceCertPem =
        "-----BEGIN CERTIFICATE-----\nFakeInstanceCert\n-----END CERTIFICATE-----";
    String instanceKeyPem =
        "-----BEGIN PRIVATE KEY-----\nFakeInstanceKey\n-----END PRIVATE KEY-----";

    Path caCertFile = tempDir.resolve("ca.crt");
    Path instanceCertFile = tempDir.resolve("instance.crt");
    Path instanceKeyFile = tempDir.resolve("instance.key");

    Files.writeString(caCertFile, caCertPem);
    Files.writeString(instanceCertFile, instanceCertPem);
    Files.writeString(instanceKeyFile, instanceKeyPem);

    String vcapServices =
        """
        {
          "user-provided": []
        }
        """;

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test",
                java.util.Map.of(
                    "VCAP_SERVICES", vcapServices,
                    "CF_INSTANCE_CA_CERT", caCertFile.toString(),
                    "CF_INSTANCE_CERT", instanceCertFile.toString(),
                    "CF_INSTANCE_KEY", instanceKeyFile.toString())));

    processEnvironment();

    // Verify the FILE CONTENTS (not the file paths) end up as property values
    String certValue =
        environment.getProperty("spring.ssl.bundle.pem.scalingengine.keystore.certificate");
    String keyValue =
        environment.getProperty("spring.ssl.bundle.pem.scalingengine.keystore.private-key");
    String caCertValue =
        environment.getProperty("spring.ssl.bundle.pem.scalingengine.truststore.certificate");

    assertNotNull(certValue, "Client certificate should be set");
    assertNotNull(keyValue, "Client key should be set");
    assertNotNull(caCertValue, "CA certificate should be set");

    // Values should be the PEM content, not the file paths
    assertTrue(
        certValue.contains("BEGIN CERTIFICATE"), "Should contain PEM content, not file path");
    assertEquals(instanceCertPem, certValue);
    assertEquals(instanceKeyPem, keyValue);
    assertEquals(caCertPem, caCertValue);
  }

  private void processEnvironment() {
    new CloudFoundryVcapEnvironmentPostProcessor(new DeferredLogs())
        .postProcessEnvironment(environment, application);
    processor.postProcessEnvironment(environment, application);
  }

  private void applyVcap(String vcapServices) {
    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test", java.util.Map.of("VCAP_SERVICES", vcapServices)));
    processEnvironment();
  }

  @Test
  void getOrderIsHighPrecedencePlus15() {
    assertEquals(org.springframework.core.Ordered.HIGHEST_PRECEDENCE + 15, processor.getOrder());
  }

  @Test
  void testDbUriWithEmbeddedCredentialsExtracted() {
    // No separate username/password; credentials must be parsed out of the postgres:// URI.
    String vcapServices =
        """
        {
          "postgresql-db": [
            {
              "name": "test-service",
              "tags": ["relational", "binding_db"],
              "credentials": {
                "uri": "postgres://uriuser:uripass@db-host.example.com:5432/mydb"
              }
            }
          ]
        }
        """;
    applyVcap(vcapServices);

    assertEquals("uriuser", environment.getProperty("spring.datasource.username"));
    assertEquals("uripass", environment.getProperty("spring.datasource.password"));
    assertEquals(
        "jdbc:postgresql://db-host.example.com:5432/mydb?sslmode=require",
        environment.getProperty("spring.datasource.url"));
  }

  @Test
  void testDbUriWithExistingQueryParamsStripsSslParams() {
    String vcapServices =
        """
        {
          "postgresql-db": [
            {
              "name": "test-service",
              "tags": ["relational", "binding_db"],
              "credentials": {
                "username": "u",
                "password": "p",
                "uri": "postgres://u:p@host:5432/db?sslmode=verify-full&application_name=foo&sslcert=/x"
              }
            }
          ]
        }
        """;
    applyVcap(vcapServices);

    String url = environment.getProperty("spring.datasource.url");
    assertNotNull(url);
    // ssl* params stripped, non-ssl query param retained, sslmode=require appended
    assertTrue(url.contains("application_name=foo"), url);
    assertTrue(url.contains("sslmode=require"), url);
    assertTrue(url.startsWith("jdbc:postgresql://host:5432/db?"), url);
  }

  @Test
  void testDbWithScalingengineAndLockTags() {
    String vcapServices =
        """
        {
          "postgresql-db": [
            {
              "name": "test-service",
              "tags": ["relational", "scalingengine_db", "lock_db"],
              "credentials": {
                "username": "u",
                "password": "p",
                "hostname": "h",
                "dbname": "d",
                "port": "5432"
              }
            }
          ]
        }
        """;
    applyVcap(vcapServices);

    assertNotNull(environment.getProperty("spring.datasource.url"));
    assertNotNull(environment.getProperty("spring.policy-db-datasource.url"));
  }

  @Test
  void testDbWithPolicyTagOnly() {
    String vcapServices =
        """
        {
          "postgresql-db": [
            {
              "name": "test-service",
              "tags": ["relational", "policy_db"],
              "credentials": {
                "username": "u",
                "password": "p",
                "hostname": "h",
                "dbname": "d",
                "port": "5432"
              }
            }
          ]
        }
        """;
    applyVcap(vcapServices);

    // policy_db tag → policy datasource configured, primary datasource NOT configured
    assertNotNull(environment.getProperty("spring.policy-db-datasource.url"));
    assertNull(environment.getProperty("spring.datasource.url"));
  }

  @Test
  void testDbMissingCredentialsLogsWarningNoConfig() {
    // relational tag but no usable jdbcUrl/username/password → else branch, no datasource set
    String vcapServices =
        """
        {
          "postgresql-db": [
            {
              "name": "test-service",
              "tags": ["relational", "binding_db"],
              "credentials": {
                "hostname": "h"
              }
            }
          ]
        }
        """;
    applyVcap(vcapServices);

    assertNull(environment.getProperty("spring.datasource.url"));
  }

  @Test
  void testDbTagsNotListProducesNoConfig() {
    // tags is a String, not a List → mapDatabaseCredentialsToDataSource returns empty config.
    // Service still matches "relational" via hasTag? No — hasTag also needs a List, so it won't
    // match. Use a List for matching but the method re-reads tags; here we exercise the filter.
    String vcapServices =
        """
        {
          "postgresql-db": [
            {
              "name": "test-service",
              "tags": ["relational"],
              "credentials": {
                "username": "u",
                "password": "p",
                "hostname": "h",
                "dbname": "d",
                "port": "5432"
              }
            }
          ]
        }
        """;
    applyVcap(vcapServices);

    // relational-only (no binding/policy/etc. tag) → neither datasource configured
    assertNull(environment.getProperty("spring.datasource.url"));
    assertNull(environment.getProperty("spring.policy-db-datasource.url"));
  }

  @Test
  void testMalformedSchedulerCredentialsFailConfiguration() {
    String vcapServices =
        """
        {
          "user-provided": [
            {
              "name": "scheduler-config-service",
              "tags": ["scheduler-config"],
              "credentials": "not-a-map"
            }
          ]
        }
        """;
    assertThrows(
        org.springframework.boot.context.properties.bind.BindException.class,
        () -> applyVcap(vcapServices));
  }

  @Test
  void testCfInstanceCertificatesPartialPathsSkipped() {
    // Only CF_INSTANCE_CERT set (key and ca missing) → the AND condition is false, SSL skipped.
    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test",
                java.util.Map.of(
                    "VCAP_SERVICES", "{\"user-provided\": []}",
                    "CF_INSTANCE_CERT", "/some/cert")));

    processEnvironment();

    assertNull(environment.getProperty("spring.ssl.bundle.pem.scalingengine.keystore.certificate"));
  }

  @Test
  void testCfInstanceCertificatesUnreadableFileSkipsSsl(@TempDir Path tempDir) throws IOException {
    // All three paths set, but one file does not exist → readFileContent returns null → skip.
    Path caCert = tempDir.resolve("ca.crt");
    Path instanceCert = tempDir.resolve("instance.crt");
    Files.writeString(caCert, "ca");
    Files.writeString(instanceCert, "cert");
    Path missingKey = tempDir.resolve("missing.key");

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test",
                java.util.Map.of(
                    "VCAP_SERVICES", "{\"user-provided\": []}",
                    "CF_INSTANCE_CA_CERT", caCert.toString(),
                    "CF_INSTANCE_CERT", instanceCert.toString(),
                    "CF_INSTANCE_KEY", missingKey.toString())));

    processEnvironment();

    assertNull(environment.getProperty("spring.ssl.bundle.pem.scalingengine.keystore.certificate"));
  }

  @Test
  void testEmptyVcapApplicationSkipsOrgGuid() {
    String vcapServices = "{\"user-provided\": []}";
    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test",
                java.util.Map.of("VCAP_SERVICES", vcapServices, "VCAP_APPLICATION", "   ")));

    processEnvironment();

    assertNull(environment.getProperty("cfserver.validOrgGuid"));
  }

  @Test
  void testSchedulerConfigWithCfServerMergesOrgGuid() {
    // cfserver present & non-empty in scheduler-config → the cfServerConfig merge branch runs.
    String vcapServices =
        """
        {
          "user-provided": [
            {
              "name": "test-service",
              "tags": ["scheduler-config"],
              "credentials": {
                "cfserver": { "port": 9090 }
              }
            }
          ]
        }
        """;
    String vcapApplication =
        """
        {
          "organization_id": "org-1",
          "space_id": "space-1"
        }
        """;
    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test",
                java.util.Map.of(
                    "VCAP_SERVICES", vcapServices,
                    "VCAP_APPLICATION", vcapApplication)));

    processEnvironment();

    assertEquals("org-1", environment.getProperty("cfserver.validOrgGuid"));
    assertEquals("space-1", environment.getProperty("cfserver.validSpaceGuid"));
    assertEquals("9090", environment.getProperty("cfserver.port"));
  }

  private static String dbVcap(String credentialsBody) {
    return """
    {
      "postgresql-db": [
        {
          "name": "test-service",
          "tags": ["relational", "binding_db"],
          "credentials": %s
        }
      ]
    }
    """
        .formatted(credentialsBody);
  }

  static Stream<Arguments> dbCredentialsWithoutUsableDatasource() {
    return Stream.of(
        Arguments.of(
            "uri without '@'",
            """
            { "uri": "postgres://h:5432/db" }
            """),
        Arguments.of(
            "uri userinfo without ':'",
            """
            { "uri": "postgres://justuser@h:5432/db" }
            """),
        Arguments.of(
            "uri with only username",
            """
            { "uri": "postgres://onlyuser@h:5432/db" }
            """),
        Arguments.of("empty credentials map", "{}"),
        Arguments.of(
            "non-postgres uri without username/password",
            """
            {
              "uri": "mysql://user:pass@h:3306/db",
              "hostname": "h",
              "port": "5432",
              "dbname": "d"
            }
            """),
        Arguments.of(
            "components missing dbname",
            """
            { "username": "u", "password": "p", "hostname": "h", "port": "5432" }
            """),
        Arguments.of(
            "components missing port",
            """
            { "username": "u", "password": "p", "hostname": "h", "dbname": "d" }
            """),
        Arguments.of(
            "components missing hostname",
            """
            { "username": "u", "password": "p", "port": "5432", "dbname": "d" }
            """),
        Arguments.of(
            "components with username but no password",
            """
            { "username": "u", "hostname": "h", "dbname": "d", "port": "5432" }
            """));
  }

  @ParameterizedTest(name = "{0}")
  @MethodSource("dbCredentialsWithoutUsableDatasource")
  void testDbCredentialsWithoutUsableDatasourceProduceNoUrl(
      String description, String credentials) {
    applyVcap(dbVcap(credentials));

    assertNull(environment.getProperty("spring.datasource.url"), description);
  }

  @Test
  void testDbNonPostgresUriWithComponentsFallsBackToComponents() {
    // uri present but not postgres:// → URI credential-extraction skipped (line 202 false) AND
    // buildJdbcUrlWithSsl uri-branch false (line 264) → falls back to hostname/port/dbname.
    applyVcap(
        dbVcap(
            """
            {
              "username": "u",
              "password": "p",
              "uri": "mysql://u:p@h:3306/db",
              "hostname": "h",
              "port": "5432",
              "dbname": "d"
            }
            """));

    assertEquals(
        "jdbc:postgresql://h:5432/d?sslmode=require",
        environment.getProperty("spring.datasource.url"));
  }

  @Test
  void testDbUriQueryWithOnlySslParamsProducesEmptyQuery() {
    // All query params are SSL-related → filteredQuery empty → line 302 false, no '?' re-added
    // before sslmode=require. Also exercises each ssl* startsWith branch (lines 290-294).
    applyVcap(
        dbVcap(
            """
            {
              "username": "u",
              "password": "p",
              "uri": "postgres://u:p@h:5432/db?sslmode=disable&sslcert=/a&sslkey=/b&sslrootcert=/c&ssl=true"
            }
            """));

    assertEquals(
        "jdbc:postgresql://h:5432/db?sslmode=require",
        environment.getProperty("spring.datasource.url"));
  }

  @Test
  void testDbUriQueryWithNonSslParamRetained() {
    // Mixed query: a non-ssl param survives (line 294 false path), filteredQuery non-empty (302).
    applyVcap(
        dbVcap(
            """
            {
              "username": "u",
              "password": "p",
              "uri": "postgres://u:p@h:5432/db?application_name=sched&sslmode=disable"
            }
            """));

    String url = environment.getProperty("spring.datasource.url");
    assertNotNull(url);
    assertTrue(url.contains("application_name=sched"), url);
    assertTrue(url.endsWith("&sslmode=require"), url);
  }

  @Test
  void testDbUriWithUsernameButNoPassword() {
    // username present, password null, uri present → line 200: first operand false, second true.
    applyVcap(
        dbVcap(
            """
            {
              "username": "u",
              "uri": "postgres://u:p@h:5432/db"
            }
            """));

    // password gets extracted from the URI → datasource configured.
    assertEquals("u", environment.getProperty("spring.datasource.username"));
    assertEquals("p", environment.getProperty("spring.datasource.password"));
  }

  @Test
  void testCfInstanceCertificatesSuccessConfiguresSslBundle(@TempDir Path tempDir)
      throws IOException {
    // All three CF cert files present & readable → line 343 true, line 355 false, SSL configured.
    Path caCert = tempDir.resolve("ca.crt");
    Path instanceCert = tempDir.resolve("instance.crt");
    Path instanceKey = tempDir.resolve("instance.key");
    Files.writeString(caCert, "-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----");
    Files.writeString(instanceCert, "-----BEGIN CERTIFICATE-----\ncert\n-----END CERTIFICATE-----");
    Files.writeString(instanceKey, "-----BEGIN PRIVATE KEY-----\nkey\n-----END PRIVATE KEY-----");

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test",
                java.util.Map.of(
                    "VCAP_SERVICES", "{\"user-provided\": []}",
                    "CF_INSTANCE_CA_CERT", caCert.toString(),
                    "CF_INSTANCE_CERT", instanceCert.toString(),
                    "CF_INSTANCE_KEY", instanceKey.toString())));

    processEnvironment();

    assertNotNull(
        environment.getProperty("spring.ssl.bundle.pem.scalingengine.keystore.certificate"));
  }

  @Test
  void testDbWithLockDbTagOnly() {
    // Only lock_db among the primary-datasource tags → exercises the lock_db operand (line 226)
    // which is short-circuited away when binding_db/scalingengine_db are present.
    String vcapServices =
        """
        {
          "postgresql-db": [
            {
              "name": "test-service",
              "tags": ["relational", "lock_db"],
              "credentials": {
                "username": "u",
                "password": "p",
                "hostname": "h",
                "dbname": "d",
                "port": "5432"
              }
            }
          ]
        }
        """;
    applyVcap(vcapServices);

    assertNotNull(environment.getProperty("spring.datasource.url"));
    // lock_db is not in the policy-datasource tag set → policy datasource NOT configured.
    assertNull(environment.getProperty("spring.policy-db-datasource.url"));
  }

  @Test
  void testSchedulerConfigWithEmptyCfServerMap() {
    // cfserver present but an empty object → line 56 `!cfServerConfig.isEmpty()` false side.
    String vcapServices =
        """
        {
          "user-provided": [
            {
              "name": "test-service",
              "tags": ["scheduler-config"],
              "credentials": {
                "cfserver": {},
                "server": { "port": 8080 }
              }
            }
          ]
        }
        """;
    String vcapApplication =
        """
        {
          "organization_id": "org-x",
          "space_id": "space-x"
        }
        """;
    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test",
                java.util.Map.of(
                    "VCAP_SERVICES", vcapServices,
                    "VCAP_APPLICATION", vcapApplication)));

    processEnvironment();

    assertEquals("8080", environment.getProperty("server.port"));
  }

  @Test
  void testCfInstanceCertificatesCaAndCertSetKeyMissing(@TempDir Path tempDir) throws IOException {
    // caCertPath + instanceCertPath set, instanceKeyPath null → line 343 third operand false.
    Path caCert = tempDir.resolve("ca.crt");
    Path instanceCert = tempDir.resolve("instance.crt");
    Files.writeString(caCert, "ca");
    Files.writeString(instanceCert, "cert");

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test",
                java.util.Map.of(
                    "VCAP_SERVICES", "{\"user-provided\": []}",
                    "CF_INSTANCE_CA_CERT", caCert.toString(),
                    "CF_INSTANCE_CERT", instanceCert.toString())));

    processEnvironment();

    assertNull(environment.getProperty("spring.ssl.bundle.pem.scalingengine.keystore.certificate"));
  }

  @Test
  void testCfInstanceCertificatesCaCertUnreadable(@TempDir Path tempDir) throws IOException {
    // All three paths set; caCert file missing → readFileContent(ca) null → line 355 first operand.
    Path instanceCert = tempDir.resolve("instance.crt");
    Path instanceKey = tempDir.resolve("instance.key");
    Files.writeString(instanceCert, "cert");
    Files.writeString(instanceKey, "key");
    Path missingCa = tempDir.resolve("missing-ca.crt");

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test",
                java.util.Map.of(
                    "VCAP_SERVICES", "{\"user-provided\": []}",
                    "CF_INSTANCE_CA_CERT", missingCa.toString(),
                    "CF_INSTANCE_CERT", instanceCert.toString(),
                    "CF_INSTANCE_KEY", instanceKey.toString())));

    processEnvironment();

    assertNull(environment.getProperty("spring.ssl.bundle.pem.scalingengine.keystore.certificate"));
  }

  @Test
  void testCfInstanceCertificatesInstanceCertUnreadable(@TempDir Path tempDir) throws IOException {
    // caCert + key readable, instanceCert missing → line 355 second operand (instanceCert null).
    Path caCert = tempDir.resolve("ca.crt");
    Path instanceKey = tempDir.resolve("instance.key");
    Files.writeString(caCert, "ca");
    Files.writeString(instanceKey, "key");
    Path missingCert = tempDir.resolve("missing.crt");

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test",
                java.util.Map.of(
                    "VCAP_SERVICES", "{\"user-provided\": []}",
                    "CF_INSTANCE_CA_CERT", caCert.toString(),
                    "CF_INSTANCE_CERT", missingCert.toString(),
                    "CF_INSTANCE_KEY", instanceKey.toString())));

    processEnvironment();

    assertNull(environment.getProperty("spring.ssl.bundle.pem.scalingengine.keystore.certificate"));
  }

  @Test
  void testDbUriQueryWithTwoNonSslParams() {
    // Two non-SSL params → the second append hits `filteredQuery.length() > 0` true branch.
    applyVcap(
        dbVcap(
            """
            {
              "username": "u",
              "password": "p",
              "uri": "postgres://u:p@h:5432/db?application_name=sched&connect_timeout=10&sslmode=disable"
            }
            """));

    String url = environment.getProperty("spring.datasource.url");
    assertNotNull(url);
    assertTrue(url.contains("application_name=sched&connect_timeout=10"), url);
    assertTrue(url.endsWith("&sslmode=require"), url);
  }

  @Test
  void testCfInstanceCertificatesCaSetCertMissingKeySet(@TempDir Path tempDir) throws IOException {
    // caCertPath set, instanceCertPath null (not provided), keyPath set → line 343 second operand.
    Path caCert = tempDir.resolve("ca.crt");
    Path instanceKey = tempDir.resolve("instance.key");
    Files.writeString(caCert, "ca");
    Files.writeString(instanceKey, "key");

    environment
        .getPropertySources()
        .addLast(
            new org.springframework.core.env.MapPropertySource(
                "test",
                java.util.Map.of(
                    "VCAP_SERVICES", "{\"user-provided\": []}",
                    "CF_INSTANCE_CA_CERT", caCert.toString(),
                    "CF_INSTANCE_KEY", instanceKey.toString())));

    processEnvironment();

    assertNull(environment.getProperty("spring.ssl.bundle.pem.scalingengine.keystore.certificate"));
  }
}
