package org.cloudfoundry.autoscaler.scheduler.conf;

import lombok.Data;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.context.annotation.Configuration;

@Data
@Configuration
@ConfigurationProperties(prefix = "cfserver")
public class CfServerConfiguration {
  private String validOrgGuid;
  private String validSpaceGuid;
  private HealthServer healthserver;
  private String systemCertPath;
  private String instanceCertPath;
  private VerificationMode xfccVerificationMode = VerificationMode.ENFORCE;

  public enum VerificationMode {
    ENFORCE,
    SHADOW,
    OFF
  }

  @Data
  public static class HealthServer {
    private String username;
    private String password;
  }
}
