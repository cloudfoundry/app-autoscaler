package org.cloudfoundry.autoscaler.scheduler;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.ImportResource;

import springfox.documentation.builders.PathSelectors;
import springfox.documentation.builders.RequestHandlerSelectors;
import springfox.documentation.spi.DocumentationType;
import springfox.documentation.spring.web.plugins.Docket;
import springfox.documentation.swagger2.annotations.EnableSwagger2;

@EnableSwagger2
@SpringBootApplication
@ImportResource("classpath:applicationContext.xml")
public class SchedulerApplication {

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
