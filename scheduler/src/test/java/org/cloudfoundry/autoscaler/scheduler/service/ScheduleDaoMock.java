package org.cloudfoundry.autoscaler.scheduler.service;

import org.cloudfoundry.autoscaler.scheduler.dao.ScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.dao.ScheduleDaoImpl;
import org.mockito.Mockito;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Primary;
import org.springframework.context.annotation.Profile;

@Profile("ScheduleDaoMock")
@Configuration
public class ScheduleDaoMock {
	ScheduleDao scheduleDao = new ScheduleDaoImpl();

	@Bean
	@Primary
	public ScheduleDao scheduleDao() {

		return Mockito.spy(scheduleDao);
	}
}
