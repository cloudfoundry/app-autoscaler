package org.cloudfoundry.autoscaler.scheduler.conf;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.util.List;
import java.util.Map;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.env.EnvironmentPostProcessor;
import org.springframework.core.Ordered;
import org.springframework.core.env.ConfigurableEnvironment;
import org.springframework.core.env.MapPropertySource;
import org.springframework.core.env.PropertySource;

public class CloudFoundryConfigurationProcessor implements EnvironmentPostProcessor, Ordered {
  private static final Logger logger =
      LoggerFactory.getLogger(CloudFoundryConfigurationProcessor.class);

  private static final String VCAP_SERVICES = "VCAP_SERVICES";
  private static final String VCAP_APPLICATION = "VCAP_APPLICATION";
  private static final String SCHEDULER_CONFIG_TAG = "scheduler-config";
  private static final String DATABASE_TAG = "relational";
  private static final ObjectMapper objectMapper = new ObjectMapper();

  @Override
  public int getOrder() {
    // Run after ConfigFileApplicationListener (HIGHEST_PRECEDENCE + 10) to override application.yml
    return Ordered.HIGHEST_PRECEDENCE + 15;
  }

  @Override
  public void postProcessEnvironment(
      ConfigurableEnvironment environment, SpringApplication application) {
    logger.info("CloudFoundryConfigurationProcessor.postProcessEnvironment() called");

    try {
      String vcapServices = environment.getProperty(VCAP_SERVICES);
      if (vcapServices == null || vcapServices.trim().isEmpty()) {
        logger.warn(
            "VCAP_SERVICES not found or empty, skipping Cloud Foundry configuration override");
        return;
      }
      logger.info("VCAP_SERVICES found, processing Cloud Foundry configuration");
      Map<String, Object> allConfigs = new java.util.HashMap<>();

      // Process scheduler-config service first
      Map<String, Object> schedulerConfig = extractSchedulerConfig(vcapServices);
      if (schedulerConfig != null && !schedulerConfig.isEmpty()) {
        logger.info(
            "Found scheduler-config service in VCAP_SERVICES, applying configuration overrides");
        allConfigs.putAll(schedulerConfig);
      }

      // Process VCAP_APPLICATION for org GUID (this should override scheduler-config)
      Map<String, Object> vcapAppConfig = extractVcapApplicationConfig(environment);
      if (vcapAppConfig != null && !vcapAppConfig.isEmpty()) {
        logger.info("Found VCAP_APPLICATION, applying cfserver configuration overrides");
        allConfigs.putAll(vcapAppConfig);
      }

      // Process database services
      Map<String, Object> databaseConfigs = extractDatabaseConfigs(vcapServices);
      if (databaseConfigs != null && !databaseConfigs.isEmpty()) {
        logger.info("Found database services in VCAP_SERVICES, applying datasource configurations");
        allConfigs.putAll(databaseConfigs);
      }

      // Process CF instance certificates for SSL configuration
      Map<String, Object> sslConfigs = extractCfInstanceCertificates(environment);
      if (sslConfigs != null && !sslConfigs.isEmpty()) {
        logger.info("Found CF instance certificates, applying SSL configuration");
        allConfigs.putAll(sslConfigs);
      }

      if (!allConfigs.isEmpty()) {
        Map<String, Object> flattenedConfig = flattenConfiguration("", allConfigs);
        PropertySource<?> cloudFoundryPropertySource =
            new MapPropertySource("cloudFoundryConfig", flattenedConfig);
        environment.getPropertySources().addFirst(cloudFoundryPropertySource);

        logger.info(
            "Applied {} configuration properties from Cloud Foundry services",
            flattenedConfig.size());
      } else {
        logger.debug("No relevant Cloud Foundry services found in VCAP_SERVICES");
      }
    } catch (Exception e) {
      logger.error("Failed to process Cloud Foundry configuration from VCAP_SERVICES", e);
    }
  }

  private Map<String, Object> extractVcapApplicationConfig(ConfigurableEnvironment environment) {
    try {
      String vcapApplication = environment.getProperty(VCAP_APPLICATION);

      if (vcapApplication == null || vcapApplication.trim().isEmpty()) {
        logger.debug("VCAP_APPLICATION not found or empty, skipping org GUID extraction");
        return null;
      }

      TypeReference<Map<String, Object>> typeRef = new TypeReference<Map<String, Object>>() {};
      Map<String, Object> vcapApp = objectMapper.readValue(vcapApplication, typeRef);

      Map<String, Object> config = new java.util.HashMap<>();
      boolean foundConfig = false;

      // Extract organization_id
      Object organizationId = vcapApp.get("organization_id");
      if (organizationId instanceof String && !((String) organizationId).trim().isEmpty()) {
        String orgGuid = (String) organizationId;
        logger.info(
            "Setting cfserver.validOrgGuid from VCAP_APPLICATION organization_id: {}", orgGuid);

        // Create nested structure to match scheduler config format
        Map<String, Object> cfserverConfig = (Map<String, Object>) config.get("cfserver");
        if (cfserverConfig == null) {
          cfserverConfig = new java.util.HashMap<>();
          config.put("cfserver", cfserverConfig);
        }
        cfserverConfig.put("validOrgGuid", orgGuid);
        foundConfig = true;
      } else {
        logger.warn("organization_id not found or empty in VCAP_APPLICATION");
      }

      // Extract space_id
      Object spaceId = vcapApp.get("space_id");
      if (spaceId instanceof String && !((String) spaceId).trim().isEmpty()) {
        String spaceGuid = (String) spaceId;
        logger.info(
            "Setting cfserver.validSpaceGuid from VCAP_APPLICATION space_id: {}", spaceGuid);

        // Create nested structure to match scheduler config format
        Map<String, Object> cfserverConfig = (Map<String, Object>) config.get("cfserver");
        if (cfserverConfig == null) {
          cfserverConfig = new java.util.HashMap<>();
          config.put("cfserver", cfserverConfig);
        }
        cfserverConfig.put("validSpaceGuid", spaceGuid);
        foundConfig = true;
      } else {
        logger.warn("space_id not found or empty in VCAP_APPLICATION");
      }

      return foundConfig ? config : null;
    } catch (Exception e) {
      logger.error("Failed to parse VCAP_APPLICATION JSON", e);
      return null;
    }
  }

  private Map<String, Object> extractSchedulerConfig(String vcapServices) {
    try {
      TypeReference<Map<String, List<Map<String, Object>>>> typeRef =
          new TypeReference<Map<String, List<Map<String, Object>>>>() {};
      Map<String, List<Map<String, Object>>> services =
          objectMapper.readValue(vcapServices, typeRef);

      return services.values().stream()
          .flatMap(List::stream)
          .filter(this::hasSchedulerConfigTag)
          .findFirst()
          .map(this::extractCredentialsFromService)
          .orElse(null);
    } catch (Exception e) {
      logger.error("Failed to parse VCAP_SERVICES JSON", e);
      return null;
    }
  }

  private Map<String, Object> extractDatabaseConfigs(String vcapServices) {
    try {
      TypeReference<Map<String, List<Map<String, Object>>>> typeRef =
          new TypeReference<Map<String, List<Map<String, Object>>>>() {};
      Map<String, List<Map<String, Object>>> services =
          objectMapper.readValue(vcapServices, typeRef);

      Map<String, Object> databaseConfigs = new java.util.HashMap<>();

      services.values().stream()
          .flatMap(List::stream)
          .filter(this::hasDatabaseTag)
          .forEach(
              service -> {
                Map<String, Object> credentialsMap = extractCredentialsFromService(service);
                if (!credentialsMap.isEmpty()) {
                  Map<String, Object> datasourceConfig =
                      mapDatabaseCredentialsToDataSource(credentialsMap, service);
                  databaseConfigs.putAll(datasourceConfig);
                }
              });

      return databaseConfigs;
    } catch (Exception e) {
      logger.error("Failed to parse VCAP_SERVICES JSON for database services", e);
      return Map.of();
    }
  }

  private Map<String, Object> mapDatabaseCredentialsToDataSource(
      Map<String, Object> credentials, Map<String, Object> service) {
    Map<String, Object> config = new java.util.HashMap<>();

    // Get service tags to determine which datasources to configure
    Object tags = service.get("tags");
    if (!(tags instanceof List)) {
      return config;
    }

    @SuppressWarnings("unchecked")
    List<String> tagList = (List<String>) tags;

    // Build JDBC URL from credentials with SSL support
    String jdbcUrl = buildJdbcUrlWithSsl(credentials);

    // Extract username and password from URI if not provided separately
    String username = (String) credentials.get("username");
    String password = (String) credentials.get("password");
    String uri = (String) credentials.get("uri");

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

  private String buildJdbcUrlWithSsl(Map<String, Object> credentials) {
    String uri = (String) credentials.get("uri");
    String hostname = (String) credentials.get("hostname");
    Object portObj = credentials.get("port");
    String dbname = (String) credentials.get("dbname");

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

        // Filter out SSL-related parameters
        String[] queryParams = queryPart.split("&");
        StringBuilder filteredQuery = new StringBuilder();
        for (String param : queryParams) {
          if (!param.toLowerCase().startsWith("ssl")
              && !param.toLowerCase().startsWith("sslmode")
              && !param.toLowerCase().startsWith("sslcert")
              && !param.toLowerCase().startsWith("sslkey")
              && !param.toLowerCase().startsWith("sslrootcert")) {
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
    } else if (hostname != null && portObj != null && dbname != null) {
      // Build from individual components
      String port = portObj.toString();
      baseUrl = String.format("jdbc:postgresql://%s:%s/%s", hostname, port, dbname);
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

  private boolean hasSchedulerConfigTag(Map<String, Object> service) {
    Object tags = service.get("tags");
    if (tags instanceof List) {
      @SuppressWarnings("unchecked")
      List<String> tagList = (List<String>) tags;
      return tagList.contains(SCHEDULER_CONFIG_TAG);
    }
    return false;
  }

  private boolean hasDatabaseTag(Map<String, Object> service) {
    Object tags = service.get("tags");
    if (tags instanceof List) {
      @SuppressWarnings("unchecked")
      List<String> tagList = (List<String>) tags;
      return tagList.contains(DATABASE_TAG);
    }
    return false;
  }

  private Map<String, Object> extractCfInstanceCertificates(ConfigurableEnvironment environment) {
    Map<String, Object> sslConfig = new java.util.HashMap<>();

    try {
      String caCert = environment.getProperty("CF_INSTANCE_CA_CERT");
      String instanceCert = environment.getProperty("CF_INSTANCE_CERT");
      String instanceKey = environment.getProperty("CF_INSTANCE_KEY");

      if (caCert != null && instanceCert != null && instanceKey != null) {
        logger.info("Found CF instance certificates, configuring SSL bundle");

        // Configure SSL bundle for the scalingengine client
        Map<String, Object> sslBundle = new java.util.HashMap<>();
        Map<String, Object> pemBundle = new java.util.HashMap<>();
        Map<String, Object> scalingengineBundle = new java.util.HashMap<>();

        // Configure keystore (client certificate and private key)
        Map<String, Object> keystore = new java.util.HashMap<>();
        keystore.put("certificate", instanceCert);
        keystore.put("private-key", instanceKey);
        scalingengineBundle.put("keystore", keystore);

        // Configure truststore (CA certificate)
        Map<String, Object> truststore = new java.util.HashMap<>();
        truststore.put("certificate", caCert);
        scalingengineBundle.put("truststore", truststore);

        pemBundle.put("scalingengine", scalingengineBundle);
        sslBundle.put("pem", pemBundle);

        Map<String, Object> springConfig = new java.util.HashMap<>();
        Map<String, Object> sslBundleConfig = new java.util.HashMap<>();
        sslBundleConfig.put("bundle", sslBundle);
        springConfig.put("ssl", sslBundleConfig);
        sslConfig.put("spring", springConfig);

        logger.info("Successfully configured SSL bundle with CF instance certificates");
      } else {
        logger.info("CF instance certificates not found, SSL bundle will not be configured");
      }
    } catch (Exception e) {
      logger.error("Error processing CF instance certificates: " + e.getMessage(), e);
    }

    return sslConfig;
  }

  private Map<String, Object> flattenConfiguration(String prefix, Map<String, Object> config) {
    Map<String, Object> flattened = new java.util.HashMap<>();

    config.forEach(
        (key, value) -> {
          String fullKey = prefix.isEmpty() ? key : prefix + "." + key;

          if (value instanceof Map) {
            @SuppressWarnings("unchecked")
            Map<String, Object> nestedMap = (Map<String, Object>) value;
            flattened.putAll(flattenConfiguration(fullKey, nestedMap));
          } else {
            flattened.put(fullKey, value);
          }
        });

    return flattened;
  }

  /**
   * Helper method to extract credentials from a service map.
   *
   * @param service The service map containing credentials
   * @return The credentials map, or empty map if not found or invalid
   */
  private Map<String, Object> extractCredentialsFromService(Map<String, Object> service) {
    Object credentials = service.get("credentials");
    if (credentials instanceof Map) {
      @SuppressWarnings("unchecked")
      Map<String, Object> credentialsMap = (Map<String, Object>) credentials;
      return credentialsMap;
    }
    return Map.<String, Object>of();
  }
}
