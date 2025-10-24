package org.cloudfoundry.autoscaler.scheduler.conf;

import lombok.Getter;
import lombok.Setter;
import org.apache.catalina.connector.Connector;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.boot.web.embedded.tomcat.TomcatServletWebServerFactory;
import org.springframework.boot.web.server.WebServerFactoryCustomizer;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

// TODO: may be move this to a better place e.g. CfServerConfiguration.java
@Configuration
@ConfigurationProperties(prefix = "server.http")
@Setter
@Getter
public class CfHttpConfiguration {

  private int port;

  @Bean
  public WebServerFactoryCustomizer<TomcatServletWebServerFactory> httpConnectorCustomizer() {
    if (port == 0) {
      return factory -> {};
    }
    return factory -> {
      Connector connector = new Connector(TomcatServletWebServerFactory.DEFAULT_PROTOCOL);
      connector.setPort(port);
      connector.setSecure(false);
      factory.addAdditionalTomcatConnectors(connector);
    };
  }
}
