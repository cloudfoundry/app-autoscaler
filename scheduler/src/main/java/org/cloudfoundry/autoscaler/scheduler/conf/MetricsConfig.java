package org.cloudfoundry.autoscaler.scheduler.conf;

import com.sun.net.httpserver.BasicAuthenticator;
import com.sun.net.httpserver.HttpsConfigurator;
import io.prometheus.client.exporter.HTTPServer;
import io.prometheus.client.exporter.HTTPServer.Builder;
import java.io.IOException;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.ssl.NoSuchSslBundleException;
import org.springframework.boot.ssl.SslBundles;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class MetricsConfig {
  private Logger logger = LoggerFactory.getLogger(this.getClass());

  @Bean(destroyMethod = "close")
  HTTPServer metricsServer(MetricsConfiguration config, SslBundles sslBundles) throws IOException {
    Builder builder = new Builder().withPort(config.getPort());

    try {
      var sslBundle = sslBundles.getBundle("healthendpoint");
      builder.withHttpsConfigurator(new HttpsConfigurator(sslBundle.createSslContext()));
    } catch (NoSuchSslBundleException e) {
      logger.warn("Starting plain-text (non-TLS) health endpoint server");
    }

    // only for /health endpoint
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
