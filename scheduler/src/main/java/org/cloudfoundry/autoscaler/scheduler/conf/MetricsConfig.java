package org.cloudfoundry.autoscaler.scheduler.conf;

import com.sun.net.httpserver.BasicAuthenticator;
import io.prometheus.client.exporter.HTTPServer;
import io.prometheus.client.exporter.HTTPServer.Builder;
import java.io.IOException;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class MetricsConfig {

  @Bean(destroyMethod = "close")
  HTTPServer metricsServer(MetricsConfiguration config) throws IOException {
    Builder builder = new Builder().withPort(config.getPort());
    if (config.isBasicAuthEnabled()) {
      builder.withAuthenticator(
          new BasicAuthenticator("/") {
            @Override
            public boolean checkCredentials(String username, String password) {
              return config.getUsername().equals(username) && config.getPassword().equals(password);
            }
          });
    }
    return builder.build();
  }
}
