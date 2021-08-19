package org.cloudfoundry.autoscaler.scheduler.conf;

import com.sun.net.httpserver.HttpContext;
import com.sun.net.httpserver.HttpServer;
import io.prometheus.client.CollectorRegistry;
import io.prometheus.client.exporter.HTTPServer;
import java.io.IOException;
import java.net.InetSocketAddress;
import java.util.stream.Stream;
import org.cloudfoundry.autoscaler.scheduler.conf.MetricsConfiguration.AuthConfig;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class MetricsConfig {

  @Bean
  HttpServer metricsServer(MetricsConfiguration config) throws IOException {
    HttpServer server = HttpServer.create(new InetSocketAddress(config.getPort()), 3);
    AuthConfig auth = config.getAuth();
    Stream<HttpContext> contexts =
        Stream.of(
            server.createContext("/"),
            server.createContext("/metrics"),
            server.createContext("/-/healthy"));
    if (config.isBasicAuthEnabled()) {
      MetricsBasicAuthentication authenticator =
          new MetricsBasicAuthentication(auth.getRealm(), auth.getUsers());
      contexts.forEach(ctx -> ctx.setAuthenticator(authenticator));
    }
    return server;
  }

  @Bean(destroyMethod = "stop")
  HTTPServer metricsServer(HttpServer server) throws IOException {
    return new HTTPServer(server, CollectorRegistry.defaultRegistry, false);
  }
}
