package org.cloudfoundry.autoscaler.scheduler.util;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.util.List;
import java.util.Map;
import java.util.Optional;

/**
 * Shared utility for parsing VCAP_SERVICES JSON and locating services by tag. Used by both {@link
 * org.cloudfoundry.autoscaler.scheduler.conf.FipsSecurityProviderConfig} (pre-Spring) and {@link
 * org.cloudfoundry.autoscaler.scheduler.conf.CloudFoundryConfigurationProcessor} (Spring
 * EnvironmentPostProcessor).
 */
public class VcapServicesHelper {

  public static final String SCHEDULER_CONFIG_TAG = "scheduler-config";
  public static final String DATABASE_TAG = "relational";

  private static final ObjectMapper objectMapper = new ObjectMapper();
  private static final TypeReference<Map<String, List<Map<String, Object>>>> VCAP_TYPE_REF =
      new TypeReference<>() {};

  private VcapServicesHelper() {}

  /** Parses VCAP_SERVICES JSON into a typed map. */
  public static Map<String, List<Map<String, Object>>> parseVcapServices(String vcapServicesJson)
      throws Exception {
    return objectMapper.readValue(vcapServicesJson, VCAP_TYPE_REF);
  }

  /** Returns true if the service entry has a "tags" list containing the given tag. */
  @SuppressWarnings("unchecked")
  public static boolean hasTag(Map<String, Object> service, String tag) {
    Object tags = service.get("tags");
    if (tags instanceof List) {
      return ((List<String>) tags).contains(tag);
    }
    return false;
  }

  /**
   * Finds the first service in VCAP_SERVICES JSON that has the given tag.
   *
   * @return the service map, or empty if not found or JSON is invalid
   */
  public static Optional<Map<String, Object>> findServiceByTag(
      String vcapServicesJson, String tag) {
    if (vcapServicesJson == null || vcapServicesJson.trim().isEmpty()) {
      return Optional.empty();
    }
    try {
      Map<String, List<Map<String, Object>>> services = parseVcapServices(vcapServicesJson);
      return services.values().stream()
          .flatMap(List::stream)
          .filter(service -> hasTag(service, tag))
          .findFirst();
    } catch (Exception e) {
      return Optional.empty();
    }
  }
}
