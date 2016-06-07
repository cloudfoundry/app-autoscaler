package org.cloudfoundry.autoscaler.scheduler;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.annotation.ImportResource;

/**
 * 
 *
 */
@SpringBootApplication
@ImportResource("classpath:applicationContext.xml")
public class SchedulerApplication {

	public static void main(String[] args) {
		SpringApplication.run(SchedulerApplication.class, args);

	}

}
