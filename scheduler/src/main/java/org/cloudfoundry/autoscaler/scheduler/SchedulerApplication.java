package org.cloudfoundry.autoscaler.scheduler;

import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.Logger;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.actuate.autoconfigure.AuditAutoConfiguration;
import org.springframework.boot.actuate.autoconfigure.EndpointAutoConfiguration;
import org.springframework.boot.actuate.autoconfigure.EndpointWebMvcAutoConfiguration;
import org.springframework.boot.actuate.autoconfigure.HealthIndicatorAutoConfiguration;
import org.springframework.boot.actuate.autoconfigure.InfoContributorAutoConfiguration;
import org.springframework.boot.actuate.autoconfigure.ManagementServerPropertiesAutoConfiguration;
import org.springframework.boot.actuate.autoconfigure.MetricExportAutoConfiguration;
import org.springframework.boot.actuate.autoconfigure.MetricRepositoryAutoConfiguration;
import org.springframework.boot.actuate.autoconfigure.PublicMetricsAutoConfiguration;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.autoconfigure.aop.AopAutoConfiguration;
import org.springframework.boot.autoconfigure.context.ConfigurationPropertiesAutoConfiguration;
import org.springframework.boot.autoconfigure.context.PropertyPlaceholderAutoConfiguration;
import org.springframework.boot.autoconfigure.gson.GsonAutoConfiguration;
import org.springframework.boot.autoconfigure.info.ProjectInfoAutoConfiguration;
import org.springframework.boot.autoconfigure.jackson.JacksonAutoConfiguration;
import org.springframework.boot.autoconfigure.jdbc.DataSourceAutoConfiguration;
import org.springframework.boot.autoconfigure.jdbc.DataSourceTransactionManagerAutoConfiguration;
import org.springframework.boot.autoconfigure.jdbc.JdbcTemplateAutoConfiguration;
import org.springframework.boot.autoconfigure.transaction.jta.JtaAutoConfiguration;
import org.springframework.boot.autoconfigure.validation.ValidationAutoConfiguration;
import org.springframework.boot.autoconfigure.web.WebClientAutoConfiguration;
import org.springframework.boot.context.event.ApplicationReadyEvent;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.ImportResource;
import org.springframework.context.event.EventListener;

import springfox.documentation.builders.PathSelectors;
import springfox.documentation.builders.RequestHandlerSelectors;
import springfox.documentation.spi.DocumentationType;
import springfox.documentation.spring.web.plugins.Docket;
import springfox.documentation.swagger2.annotations.EnableSwagger2;

@EnableSwagger2
@SpringBootApplication(exclude = { AopAutoConfiguration.class, HealthIndicatorAutoConfiguration.class,
		AuditAutoConfiguration.class, PropertyPlaceholderAutoConfiguration.class, PublicMetricsAutoConfiguration.class,
		InfoContributorAutoConfiguration.class, WebClientAutoConfiguration.class, EndpointAutoConfiguration.class,
		ConfigurationPropertiesAutoConfiguration.class, MetricRepositoryAutoConfiguration.class,
		ProjectInfoAutoConfiguration.class, MetricExportAutoConfiguration.class, DataSourceAutoConfiguration.class,
		GsonAutoConfiguration.class, ValidationAutoConfiguration.class,
		DataSourceTransactionManagerAutoConfiguration.class, EndpointWebMvcAutoConfiguration.class,
		JacksonAutoConfiguration.class, JdbcTemplateAutoConfiguration.class, JtaAutoConfiguration.class,
		ManagementServerPropertiesAutoConfiguration.class })
@ImportResource("classpath:applicationContext.xml")
public class SchedulerApplication {
	private Logger logger = LogManager.getLogger(this.getClass());

	@EventListener
	public void onApplicationReady(ApplicationReadyEvent event) {
		logger.info("Scheduler is ready to start");
	}

	public static void main(String[] args) {
		SpringApplication.run(SchedulerApplication.class, args);
	}

	@Bean
	public Docket api() {
		return new Docket(DocumentationType.SWAGGER_2).useDefaultResponseMessages(false).select()
				.apis(RequestHandlerSelectors.basePackage("org.cloudfoundry.autoscaler.scheduler"))
				.paths(PathSelectors.any()).build();
	}

}
