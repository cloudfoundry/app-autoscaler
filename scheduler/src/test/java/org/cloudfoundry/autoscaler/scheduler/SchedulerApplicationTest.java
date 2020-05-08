package org.cloudfoundry.autoscaler.scheduler;

import static org.hamcrest.Matchers.equalToIgnoringCase;
import static org.hamcrest.Matchers.isA;
import static org.junit.Assert.assertThat;

import org.apache.commons.dbcp2.BasicDataSource;
import org.cloudfoundry.autoscaler.scheduler.util.error.DataSourceConfigurationException;
import org.junit.Rule;
import org.junit.Test;
import org.junit.rules.ExpectedException;
import org.junit.runner.RunWith;
import org.springframework.beans.factory.BeanCreationException;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.junit4.SpringRunner;

@RunWith(SpringRunner.class)
@SpringBootTest
public class SchedulerApplicationTest {
	@Autowired
	private BasicDataSource dataSource;

	@Rule
	public ExpectedException expectedEx = ExpectedException.none();

	@Test
	public void  testTomcatConnectionPoolNameCorrect() {
		assertThat(dataSource.getClass().getName(), equalToIgnoringCase("org.apache.commons.dbcp2.BasicDataSource"));
	}

	@Test
	public void testApplicationExitsWhenSchedulerDBUnreachable() {
		expectedEx.expect(BeanCreationException.class);
		expectedEx.expectCause(isA(DataSourceConfigurationException.class));
		expectedEx.expectMessage("Error creating bean with name 'dataSource': Failed to connect to datasource:dataSource");
		SchedulerApplication.main(new String[] { 
				"--spring.datasource.url=jdbc:postgresql://127.0.0.1/wrong-scheduler-db" });

	}

	@Test
	public void testApplicationExitsWhenPolicyDBUnreachable() {
		expectedEx.expect(BeanCreationException.class);
		expectedEx.expectCause(isA(DataSourceConfigurationException.class));
		expectedEx.expectMessage("Error creating bean with name 'policyDbDataSource': Failed to connect to datasource:policyDbDataSource");
		SchedulerApplication.main(new String[] { 
				"--spring.policyDbDataSource.url=jdbc:postgresql://127.0.0.1/wrong-policy-db" });


	}
}