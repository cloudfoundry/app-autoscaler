package org.cloudfoundry.autoscaler.scheduler.dao;

import org.mockito.Mockito;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Primary;
import org.springframework.context.annotation.Profile;

@Profile("ActiveScheduleDaoMock")
@Configuration
public class ActiveScheduleDaoMock {

	@Bean
	@Primary
	public ActiveScheduleDao activeScheduleDao() {
		return Mockito.mock(ActiveScheduleDao.class);
	}
}
