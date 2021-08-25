package org.cloudfoundry.autoscaler.scheduler.conf;

import lombok.Data;
import org.springframework.boot.context.properties.ConfigurationProperties;

@ConfigurationProperties(prefix = "scheduler.healthserver")
@Data
public class MetricsConfiguration {

  private int port;
}
