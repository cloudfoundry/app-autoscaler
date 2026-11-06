package org.cloudfoundry.autoscaler.scheduler.conf;

import org.springframework.boot.EnvironmentPostProcessor;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.context.properties.bind.Binder;
import org.springframework.core.Ordered;
import org.springframework.core.env.ConfigurableEnvironment;

/** Activates FIPS before bean creation for both executable and external-Tomcat deployments. */
public class FipsEnvironmentPostProcessor implements EnvironmentPostProcessor, Ordered {

  @Override
  public int getOrder() {
    return CloudFoundryConfigurationProcessor.ORDER + 1;
  }

  @Override
  public void postProcessEnvironment(
      ConfigurableEnvironment environment, SpringApplication application) {
    if (Binder.get(environment).bind("autoscaler.fips-mode", Boolean.class).orElse(false)) {
      FipsSecurityProviderConfig.initialize();
    }
  }
}
