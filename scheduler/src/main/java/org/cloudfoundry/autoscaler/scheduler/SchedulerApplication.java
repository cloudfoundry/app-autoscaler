package org.cloudfoundry.autoscaler.scheduler;

import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.Logger;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.ImportAutoConfiguration;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.context.event.ApplicationReadyEvent;
import org.springframework.cloud.client.discovery.EnableDiscoveryClient;
import org.springframework.cloud.client.serviceregistry.AutoServiceRegistrationConfiguration;
import org.springframework.cloud.consul.serviceregistry.ConsulAutoServiceRegistration;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.ImportResource;
import org.springframework.context.event.EventListener;

import springfox.documentation.builders.PathSelectors;
import springfox.documentation.builders.RequestHandlerSelectors;
import springfox.documentation.spi.DocumentationType;
import springfox.documentation.spring.web.plugins.Docket;
import springfox.documentation.swagger2.annotations.EnableSwagger2;

@EnableSwagger2
@EnableDiscoveryClient(autoRegister = false)
@SpringBootApplication
@ImportResource("classpath:applicationContext.xml")
@ImportAutoConfiguration({ AutoServiceRegistrationConfiguration.class })
public class SchedulerApplication {
	private Logger logger = LogManager.getLogger(this.getClass());

	@Autowired
	private ConsulAutoServiceRegistration autoServiceRegistration;

	@EventListener
	public void onApplicationReady(ApplicationReadyEvent event) {
		autoServiceRegistration.start();
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
