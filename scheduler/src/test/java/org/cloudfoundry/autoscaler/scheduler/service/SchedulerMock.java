package org.cloudfoundry.autoscaler.scheduler.service;

import org.mockito.Mockito;
import org.quartz.Scheduler;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Primary;
import org.springframework.context.annotation.Profile;

@Profile("SchedulerMock")
@Configuration
public class SchedulerMock {

	@Autowired
	Scheduler scheduler;

	@Bean
	@Primary
	public Scheduler scheduler() {
		return Mockito.spy(scheduler);
	}
}
