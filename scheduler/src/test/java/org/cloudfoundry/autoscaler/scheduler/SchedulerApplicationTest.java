package org.cloudfoundry.autoscaler.scheduler;

import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.Matchers.equalToIgnoringCase;

import com.zaxxer.hikari.HikariDataSource;
import org.junit.Assert;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.junit4.SpringRunner;

@RunWith(SpringRunner.class)
@SpringBootTest
public class SchedulerApplicationTest {
  @Autowired private HikariDataSource dataSource;

  @Test
  public void testTomcatConnectionPoolNameCorrect() {
    assertThat(
        dataSource.getClass().getName(), equalToIgnoringCase("com.zaxxer.hikari.HikariDataSource"));
  }

  @Test
  public void testApplicationExitsWhenSchedulerDbUnreachable() {
    Assert.assertThrows(
        org.springframework.beans.factory.BeanCreationException.class,
        () ->
            SchedulerApplication.main(
                new String[] {
                  "--spring.autoconfigure.exclude="
                      + "org.springframework.boot.actuate.autoconfigure.jdbc."
                      + "DataSourceHealthIndicatorAutoConfiguration",
                  "--spring.datasource.url=jdbc:postgresql://127.0.0.1/wrong-scheduler-db"
                }));
  }

  @Test
  public void testApplicationExitsWhenPolicyDbUnreachable() {
    Assert.assertThrows(
        org.springframework.beans.factory.BeanCreationException.class,
        () ->
            SchedulerApplication.main(
                new String[] {
                  "--spring.autoconfigure.exclude="
                      + "org.springframework.boot.actuate.autoconfigure.jdbc."
                      + "DataSourceHealthIndicatorAutoConfiguration",
                  "--spring.policy-db-datasource.url=jdbc:postgresql://127.0.0.1/wrong-policy-db"
                }));
  }
}
