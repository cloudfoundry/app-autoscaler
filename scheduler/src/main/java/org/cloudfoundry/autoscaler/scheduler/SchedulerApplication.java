package org.cloudfoundry.autoscaler.scheduler;

import org.cloudfoundry.autoscaler.scheduler.conf.FipsSecurityProviderConfig;
import org.cloudfoundry.autoscaler.scheduler.conf.MetricsConfig;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.autoconfigure.aop.AopAutoConfiguration;
import org.springframework.boot.autoconfigure.context.ConfigurationPropertiesAutoConfiguration;
import org.springframework.boot.autoconfigure.context.PropertyPlaceholderAutoConfiguration;
import org.springframework.boot.autoconfigure.info.ProjectInfoAutoConfiguration;
import org.springframework.boot.context.event.ApplicationReadyEvent;
import org.springframework.boot.context.properties.ConfigurationPropertiesScan;
import org.springframework.boot.data.jpa.autoconfigure.DataJpaRepositoriesAutoConfiguration;
import org.springframework.boot.gson.autoconfigure.GsonAutoConfiguration;
import org.springframework.boot.hibernate.autoconfigure.HibernateJpaAutoConfiguration;
import org.springframework.boot.jackson.autoconfigure.JacksonAutoConfiguration;
import org.springframework.boot.jdbc.autoconfigure.DataSourceAutoConfiguration;
import org.springframework.boot.jdbc.autoconfigure.DataSourceTransactionManagerAutoConfiguration;
import org.springframework.boot.jdbc.autoconfigure.JdbcTemplateAutoConfiguration;
import org.springframework.boot.transaction.jta.autoconfigure.JtaAutoConfiguration;
import org.springframework.boot.webclient.autoconfigure.WebClientAutoConfiguration;
import org.springframework.context.event.EventListener;

@ConfigurationPropertiesScan(basePackageClasses = MetricsConfig.class)
@SpringBootApplication(
    exclude = {
      AopAutoConfiguration.class,
      DataSourceAutoConfiguration.class,
      WebClientAutoConfiguration.class,
      ProjectInfoAutoConfiguration.class,
      ConfigurationPropertiesAutoConfiguration.class,
      GsonAutoConfiguration.class,
      PropertyPlaceholderAutoConfiguration.class,
      DataSourceTransactionManagerAutoConfiguration.class,
      JacksonAutoConfiguration.class,
      JdbcTemplateAutoConfiguration.class,
      JtaAutoConfiguration.class,
      HibernateJpaAutoConfiguration.class,
      DataJpaRepositoriesAutoConfiguration.class
    })
public class SchedulerApplication {

  private static final Logger logger = LoggerFactory.getLogger(SchedulerApplication.class);

  @EventListener
  public void onApplicationReady(ApplicationReadyEvent event) {
    logger.info("Scheduler is ready to start");
  }

  public static void main(String[] args) {
    if (FipsSecurityProviderConfig.isFipsModeEnabled()) {
      FipsSecurityProviderConfig.initialize();
      logger.info("FIPS 140-3 mode enabled");
    } else {
      logger.info("FIPS 140-3 mode is not enabled");
    }
    SpringApplication.run(SchedulerApplication.class, args);
  }
}
