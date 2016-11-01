package org.cloudfoundry.autoscaler.scheduler.service;

import org.cloudfoundry.autoscaler.scheduler.dao.RecurringScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.dao.RecurringScheduleDaoImpl;
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
	private SpecificDateScheduleDao specificDateScheduleDao = new SpecificDateScheduleDaoImpl();
	private RecurringScheduleDao recurringScheduleDao = new RecurringScheduleDaoImpl();

	@Bean
	@Primary
	public SpecificDateScheduleDao specificDateScheduleDao() {
		return Mockito.spy(specificDateScheduleDao);
	}

	@Bean
	@Primary
	public RecurringScheduleDao recurringScheduleDao() {
		return Mockito.spy(recurringScheduleDao);
	}
}
