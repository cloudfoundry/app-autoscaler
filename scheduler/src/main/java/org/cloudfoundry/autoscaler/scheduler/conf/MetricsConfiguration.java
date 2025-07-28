package org.cloudfoundry.autoscaler.scheduler.conf;

import jakarta.annotation.PostConstruct;
import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.stereotype.Component;

@ConfigurationProperties(prefix = "scheduler.healthserver")
@Data
@Component
@AllArgsConstructor
@NoArgsConstructor
public class MetricsConfiguration {
  private String username;
  private String password;
  private int port;
  private boolean basicAuthEnabled = false;

  @PostConstruct
  public void init() {
    if (this.basicAuthEnabled
        && (this.username == null
            || this.password == null
            || this.username.isEmpty()
            || this.password.isEmpty())) {
      throw new IllegalStateException("Heath Server Basic Auth Username or password is not set");
    }
  }
}
