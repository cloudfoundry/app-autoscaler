package org.cloudfoundry.autoscaler.scheduler.conf;

import static org.assertj.core.api.Assertions.assertThat;

import java.security.GeneralSecurityException;
import java.security.KeyFactory;
import java.security.cert.CertificateFactory;
import org.bouncycastle.crypto.CryptoServicesRegistrar;
import org.springframework.boot.autoconfigure.ssl.SslAutoConfiguration;
import org.springframework.boot.builder.SpringApplicationBuilder;
import org.springframework.boot.web.servlet.support.SpringBootServletInitializer;
import org.springframework.context.ConfigurableApplicationContext;
import org.springframework.mock.web.MockServletContext;

/** Exercises the WAR bootstrap and registered environment processors without starting a database. */
class FipsStartupSupport extends SpringBootServletInitializer {

  private final boolean expectFips;

  private FipsStartupSupport(boolean expectFips) {
    this.expectFips = expectFips;
    setRegisterErrorPageFilter(false);
  }

  static ConfigurableApplicationContext start(String credentials, boolean expectFips) {
    MockServletContext servletContext = new MockServletContext();
    servletContext.addInitParameter(
        "spring.config.location", "optional:classpath:/fips-startup-test.properties");
    servletContext.addInitParameter("spring.main.banner-mode", "off");
    servletContext.addInitParameter("client.ssl.protocol", "TLSv1.2");
    servletContext.addInitParameter("client.httpClientTimeout", "5");
    servletContext.addInitParameter("CF_CA_CERTS", "src/test/resources/certs/test-ca.crt");
    servletContext.addInitParameter(
        "VCAP_SERVICES",
        """
        {"user-provided": [{
          "name": "renamed-scheduler-service",
          "tags": ["scheduler-config"],
          "credentials": %s
        }]}
        """
            .formatted(credentials));
    servletContext.addInitParameter(
        "CF_INSTANCE_CERT", "src/test/resources/certs/test-scheduler.crt");
    servletContext.addInitParameter(
        "CF_INSTANCE_KEY", "src/test/resources/certs/test-scheduler.key");
    return (ConfigurableApplicationContext)
        new FipsStartupSupport(expectFips).createRootApplicationContext(servletContext);
  }

  @Override
  protected SpringApplicationBuilder configure(SpringApplicationBuilder builder) {
    return builder
        .sources(SslAutoConfiguration.class, RestClientConfig.class)
        .initializers(
            context -> {
              // Initializers run before the context refresh creates SSL bundles and other beans.
              assertThat(FipsSecurityProviderConfig.isInitialized()).isEqualTo(expectFips);
              if (expectFips) {
                assertThat(CryptoServicesRegistrar.isInApprovedOnlyMode()).isTrue();
                try {
                  assertThat(KeyFactory.getInstance("RSA").getProvider().getName())
                      .isEqualTo("BCFIPS");
                  assertThat(CertificateFactory.getInstance("X.509").getProvider().getName())
                      .isEqualTo("BCFIPS");
                } catch (GeneralSecurityException e) {
                  throw new IllegalStateException(e);
                }
              }
            });
  }
}
