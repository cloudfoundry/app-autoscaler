package org.cloudfoundry.autoscaler.scheduler.dao;

import org.apache.commons.dbcp.BasicDataSource;
import org.mockito.Mockito;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Primary;
import org.springframework.context.annotation.Profile;

@Profile("DataSourceMock")
@Configuration
public class DataSourceMock {

	@Bean
	@Primary
	public BasicDataSource basicDataSource() {
		return Mockito.mock(BasicDataSource.class);
	}

}
