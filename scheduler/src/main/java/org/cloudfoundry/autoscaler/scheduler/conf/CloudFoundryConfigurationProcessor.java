package org.cloudfoundry.autoscaler.scheduler.conf;

import java.util.List;
import java.util.Map;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.EnvironmentPostProcessor;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.cloud.CloudPlatform;
import org.springframework.boot.context.config.ConfigDataEnvironmentPostProcessor;
import org.springframework.boot.context.properties.bind.Bindable;
import org.springframework.boot.context.properties.bind.Binder;
import org.springframework.core.Ordered;
import org.springframework.core.env.ConfigurableEnvironment;
import org.springframework.core.env.MapPropertySource;
import org.springframework.core.env.PropertySource;
import org.springframework.util.StringUtils;

public class CloudFoundryConfigurationProcessor implements EnvironmentPostProcessor, Ordered {
  private static final Logger logger =
      LoggerFactory.getLogger(CloudFoundryConfigurationProcessor.class);

  // Boot's VCAP processor runs before config data; apply scheduler overrides afterwards.
  static final int ORDER = ConfigDataEnvironmentPostProcessor.ORDER + 5;

  @Override
  public int getOrder() {
    return ORDER;
  }

  @Override
  public void postProcessEnvironment(
      ConfigurableEnvironment environment, SpringApplication application) {
    if (!CloudPlatform.CLOUD_FOUNDRY.isActive(environment)) {
      return;
    }
    Map<String, ServiceBinding> services =
        Binder.get(environment)
            .bind("vcap.services", Bindable.mapOf(String.class, ServiceBinding.class))
            .orElseGet(Map::of);
    Map<String, Object> config = new java.util.HashMap<>();
    services.values().stream()
        .filter(service -> service.tags().contains("scheduler-config"))
        .findFirst()
        .ifPresent(service -> config.putAll(service.credentials()));

    if (config.keySet().stream().anyMatch(key -> key.startsWith("cfserver."))) {
      copyProperty(environment, config, "vcap.application.organization_id", "cfserver.validOrgGuid");
      copyProperty(environment, config, "vcap.application.space_id", "cfserver.validSpaceGuid");
    }

    services.values().stream()
        .filter(service -> service.tags().contains("relational"))
        .forEach(
            service ->
                config.putAll(
                    mapDatabaseCredentialsToDataSource(service.credentials(), service.tags())));
    config.putAll(extractCfInstanceCertificates(environment));

    if (!config.isEmpty()) {
      PropertySource<?> propertySource = new MapPropertySource("cloudFoundryConfig", config);
      environment.getPropertySources().addFirst(propertySource);
      logger.info("Applied {} configuration properties from Cloud Foundry services", config.size());
    }
  }

  private void copyProperty(
      ConfigurableEnvironment environment, Map<String, Object> config, String source, String target) {
    String value = environment.getProperty(source);
    if (StringUtils.hasText(value)) {
      config.put(target, value);
    }
  }

  record ServiceBinding(List<String> tags, Map<String, String> credentials) {
    ServiceBinding {
      tags = tags != null ? tags : List.of();
      credentials = credentials != null ? credentials : Map.of();
    }
  }

  private Map<String, Object> mapDatabaseCredentialsToDataSource(
      Map<String, String> credentials, List<String> tagList) {
    Map<String, Object> config = new java.util.HashMap<>();

    // Build JDBC URL from credentials with SSL support
    String jdbcUrl = buildJdbcUrlWithSsl(credentials);

    // Extract username and password from URI if not provided separately
    String username = credentials.get("username");
    String password = credentials.get("password");
    String uri = credentials.get("uri");

    if ((username == null || password == null) && uri != null) {
      // Extract from postgres://username:password@host:port/database format
      if (uri.startsWith("postgres://")) {
        try {
          String userInfo = uri.substring("postgres://".length());
          int atIndex = userInfo.indexOf('@');
          if (atIndex > 0) {
            String credentialsPart = userInfo.substring(0, atIndex);
            int colonIndex = credentialsPart.indexOf(':');
            if (colonIndex > 0) {
              username = credentialsPart.substring(0, colonIndex);
              password = credentialsPart.substring(colonIndex + 1);
            }
          }
        } catch (Exception e) {
          logger.warn(
              "Failed to parse username/password from URI: {}. Error: {}", uri, e.getMessage());
        }
      }
    }

    if (jdbcUrl != null && username != null && password != null) {
      // Configure primary datasource if binding_db, scalingengine_db, or lock_db tag is present or
      // no specific tags
      if (tagList.contains("binding_db")
          || tagList.contains("scalingengine_db")
          || tagList.contains("lock_db")) {
        config.put("spring.datasource.url", jdbcUrl);
        config.put("spring.datasource.username", username);
        config.put("spring.datasource.password", password);
        config.put("spring.datasource.driverClassName", "org.postgresql.Driver");
      }

      // Configure policy datasource if policy_db tag is present or as fallback
      if (tagList.contains("policy_db")
          || tagList.contains("binding_db")
          || tagList.contains("scalingengine_db")) {
        config.put("spring.policy-db-datasource.url", jdbcUrl);
        config.put("spring.policy-db-datasource.username", username);
        config.put("spring.policy-db-datasource.password", password);
        config.put("spring.policy-db-datasource.driverClassName", "org.postgresql.Driver");
      }

      logger.info("Configured datasources for database service with tags: {}", tagList);
    } else {
      logger.warn(
          "Missing required credentials for database configuration: jdbcUrl={}, username={},"
              + " password={}",
          jdbcUrl,
          username,
          password);
    }

    return config;
  }

  private String buildJdbcUrlWithSsl(Map<String, String> credentials) {
    String uri = credentials.get("uri");
    String hostname = credentials.get("hostname");
    String port = credentials.get("port");
    String dbname = credentials.get("dbname");

    // Build base URL
    String baseUrl = null;
    if (uri != null && uri.startsWith("postgres://")) {
      // Convert postgres:// URI to jdbc:postgresql:// format and strip credentials and SSL
      // parameters
      String cleanUri = uri.replace("postgres://", "jdbc:postgresql://");

      // Remove credentials from URI (username:password@)
      if (cleanUri.contains("@")) {
        int atIndex = cleanUri.indexOf("@");
        String protocolPart = "jdbc:postgresql://";
        String hostPart = cleanUri.substring(atIndex + 1);
        cleanUri = protocolPart + hostPart;
      }

      baseUrl = cleanUri;

      // Remove existing SSL-related query parameters
      if (baseUrl.contains("?")) {
        String[] parts = baseUrl.split("\\?", 2);
        String urlPart = parts[0];
        String queryPart = parts[1];

        // Filter out SSL-related parameters (sslmode, sslcert, sslkey, sslrootcert all start "ssl")
        String[] queryParams = queryPart.split("&");
        StringBuilder filteredQuery = new StringBuilder();
        for (String param : queryParams) {
          if (!param.toLowerCase().startsWith("ssl")) {
            if (filteredQuery.length() > 0) {
              filteredQuery.append("&");
            }
            filteredQuery.append(param);
          }
        }

        baseUrl = urlPart;
        if (filteredQuery.length() > 0) {
          baseUrl += "?" + filteredQuery.toString();
        }
      }
    } else if (hostname != null && port != null && dbname != null) {
      // Build from individual components
      baseUrl = "jdbc:postgresql://%s:%s/%s".formatted(hostname, port, dbname);
    }

    if (baseUrl == null) {
      return null;
    }

    // For Cloud Foundry, use simple SSL mode without certificate files
    StringBuilder urlBuilder = new StringBuilder(baseUrl);
    if (!baseUrl.contains("?")) {
      urlBuilder.append("?");
    } else {
      urlBuilder.append("&");
    }

    // Use require SSL mode but without client certificates for Cloud Foundry
    urlBuilder.append("sslmode=require");

    logger.info("Using SSL mode 'require' for Cloud Foundry database connection");

    return urlBuilder.toString();
  }

  private Map<String, Object> extractCfInstanceCertificates(ConfigurableEnvironment environment) {
    Map<String, Object> sslConfig = new java.util.HashMap<>();

    try {
      // CF_INSTANCE_CERT, CF_INSTANCE_KEY, and CF_INSTANCE_CA_CERT are FILE PATHS,
      // not inline PEM content. See:
      // https://docs.cloudfoundry.org/devguide/deploy-apps/instance-identity.html
      String caCertPath = environment.getProperty("CF_INSTANCE_CA_CERT");
      String instanceCertPath = environment.getProperty("CF_INSTANCE_CERT");
      String instanceKeyPath = environment.getProperty("CF_INSTANCE_KEY");

      // CF_INSTANCE_CA_CERT is optional; CF instance identity only requires the cert and key.
      if (instanceCertPath != null && instanceKeyPath != null) {
        logger.info(
            "Found CF instance certificate paths: cert={}, key={}, ca={}",
            instanceCertPath,
            instanceKeyPath,
            caCertPath);

        // Read PEM contents for the SSL bundle overrides.
        String caCert = caCertPath != null ? readFileContent(caCertPath) : null;
        String instanceCert = readFileContent(instanceCertPath);
        String instanceKey = readFileContent(instanceKeyPath);

        if ((caCertPath != null && caCert == null) || instanceCert == null || instanceKey == null) {
          logger.error("Failed to read one or more CF instance certificate files");
          return sslConfig;
        }

        String bundlePrefix = "spring.ssl.bundle.pem.scalingengine.";
        sslConfig.put(bundlePrefix + "keystore.certificate", instanceCert);
        sslConfig.put(bundlePrefix + "keystore.private-key", instanceKey);
        if (caCert != null) {
          sslConfig.put(bundlePrefix + "truststore.certificate", caCert);
        }

        logger.info("Successfully configured SSL bundle with CF instance certificates");
      } else {
        logger.info("CF instance certificates not found, SSL bundle will not be configured");
      }
    } catch (Exception e) {
      logger.error("Error processing CF instance certificates: " + e.getMessage(), e);
    }

    return sslConfig;
  }

  /** Reads PEM content from a CF_INSTANCE_* file path. */
  private String readFileContent(String path) {
    try {
      return java.nio.file.Files.readString(java.nio.file.Paths.get(path));
    } catch (Exception e) {
      logger.error("Failed to read file {}: {}", path, e.getMessage());
      return null;
    }
  }
}
