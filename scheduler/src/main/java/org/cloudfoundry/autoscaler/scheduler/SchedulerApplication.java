package org.cloudfoundry.autoscaler.scheduler;

import org.cloudfoundry.autoscaler.scheduler.conf.MetricsConfig;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.autoconfigure.aop.AopAutoConfiguration;
import org.springframework.boot.autoconfigure.context.ConfigurationPropertiesAutoConfiguration;
import org.springframework.boot.autoconfigure.context.PropertyPlaceholderAutoConfiguration;
import org.springframework.boot.autoconfigure.data.jpa.JpaRepositoriesAutoConfiguration;
import org.springframework.boot.autoconfigure.gson.GsonAutoConfiguration;
import org.springframework.boot.autoconfigure.info.ProjectInfoAutoConfiguration;
import org.springframework.boot.autoconfigure.jackson.JacksonAutoConfiguration;
import org.springframework.boot.autoconfigure.jdbc.DataSourceAutoConfiguration;
import org.springframework.boot.autoconfigure.jdbc.DataSourceTransactionManagerAutoConfiguration;
import org.springframework.boot.autoconfigure.jdbc.JdbcTemplateAutoConfiguration;
import org.springframework.boot.autoconfigure.orm.jpa.HibernateJpaAutoConfiguration;
import org.springframework.boot.autoconfigure.transaction.jta.JtaAutoConfiguration;
import org.springframework.boot.autoconfigure.web.reactive.function.client.WebClientAutoConfiguration;
import org.springframework.boot.context.event.ApplicationReadyEvent;
import org.springframework.boot.context.properties.ConfigurationPropertiesScan;
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
      JpaRepositoriesAutoConfiguration.class
    })
public class SchedulerApplication {

  private Logger logger = LoggerFactory.getLogger(this.getClass());

  @EventListener
  public void onApplicationReady(ApplicationReadyEvent event) {
    logger.info("Scheduler is ready to start");
  }

  public static void main(String[] args) {
    SpringApplication.run(SchedulerApplication.class, args);
  }
}
