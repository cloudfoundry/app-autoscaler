package org.cloudfoundry.autoscaler.scheduler;

import static org.hamcrest.CoreMatchers.containsString;
import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.Matchers.equalToIgnoringCase;
import static org.hamcrest.Matchers.isA;
import static org.junit.jupiter.api.Assertions.assertThrows;

import org.apache.commons.dbcp2.BasicDataSource;
import org.cloudfoundry.autoscaler.scheduler.util.error.DataSourceConfigurationException;
import org.junit.jupiter.api.Test;
import org.junit.runner.RunWith;
import org.springframework.beans.factory.BeanCreationException;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.junit4.SpringRunner;

@RunWith(SpringRunner.class)
@SpringBootTest
public class SchedulerApplicationTest {
  @Autowired private BasicDataSource dataSource;

  @Test
  public void testTomcatConnectionPoolNameCorrect() {
    assertThat(
        dataSource.getClass().getName(),
        equalToIgnoringCase("org.apache.commons.dbcp2.BasicDataSource"));
  }

  @Test
  public void testApplicationExitsWhenSchedulerDbUnreachable() {
      Throwable exception = assertThrows(BeanCreationException.class, () ->
          SchedulerApplication.main(
                  new String[]{
                          "--spring.autoconfigure.exclude="
                                  + "org.springframework.boot.actuate.autoconfigure.jdbc."
                                  + "DataSourceHealthIndicatorAutoConfiguration",
                          "--spring.datasource.url=jdbc:postgresql://127.0.0.1/wrong-scheduler-db"
                  }));
      assertThat(exception.getCause(), isA(DataSourceConfigurationException.class));
      assertThat(exception.getMessage(), containsString("Error creating bean with name 'dataSource': Failed to connect to datasource:dataSource"));
  }

  @Test
  public void testApplicationExitsWhenPolicyDbUnreachable() {
      Throwable exception = assertThrows(BeanCreationException.class, () ->
          SchedulerApplication.main(
                  new String[]{
                          "--spring.autoconfigure.exclude="
                                  + "org.springframework.boot.actuate.autoconfigure.jdbc."
                                  + "DataSourceHealthIndicatorAutoConfiguration",
                          "--spring.policy-db-datasource.url=jdbc:postgresql://127.0.0.1/wrong-policy-db"
                  }));
      assertThat(exception.getCause(), isA(DataSourceConfigurationException.class));
      assertThat(exception.getMessage(), containsString("Error creating bean with name 'policyDbDataSource': Failed to connect to"
              + " datasource:policyDbDataSource"));
  }
}
