package org.cloudfoundry.autoscaler.scheduler.conf;

import java.util.Map;
import lombok.Data;
import org.springframework.boot.context.properties.ConfigurationProperties;

@ConfigurationProperties(prefix = "scheduler.metrics")
@Data
public class MetricsConfiguration {

  @Data
  public static class AuthConfig {
    private Map<String, String> users;
    private String realm;
  }

  private int port;
  private boolean basicAuthEnabled = true;
  private AuthConfig auth;
}
