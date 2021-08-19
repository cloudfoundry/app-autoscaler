package org.cloudfoundry.autoscaler.scheduler.conf;

import lombok.Data;
import org.springframework.boot.context.properties.ConfigurationProperties;

@ConfigurationProperties(prefix = "scheduler.metrics")
@Data
public class MetricsConfiguration {
  private String username;
  private String password;
  private int port;
  private boolean basicAuthEnabled = true;
}
