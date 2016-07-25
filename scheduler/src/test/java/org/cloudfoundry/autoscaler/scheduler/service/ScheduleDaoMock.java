package org.cloudfoundry.autoscaler.scheduler.service;

import org.cloudfoundry.autoscaler.scheduler.dao.SpecificDateScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.dao.SpecificDateScheduleDaoImpl;
import org.mockito.Mockito;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Primary;
import org.springframework.context.annotation.Profile;

@Profile("ScheduleDaoMock")
@Configuration
public class ScheduleDaoMock {
	SpecificDateScheduleDao specificDateScheduleDao = new SpecificDateScheduleDaoImpl();

	@Bean
	@Primary
	public SpecificDateScheduleDao specificDateScheduleDao() {

		return Mockito.spy(specificDateScheduleDao);
	}
}
