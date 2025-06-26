package org.cloudfoundry.autoscaler.scheduler.dao;

import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.Matchers.is;
import static org.junit.Assert.fail;

import java.sql.SQLException;
import javax.sql.DataSource;
import org.cloudfoundry.autoscaler.scheduler.entity.ActiveScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataDbUtil;
import org.cloudfoundry.autoscaler.scheduler.util.error.DatabaseValidationException;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.Mockito;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.bean.override.mockito.MockitoSpyBean;
import org.springframework.test.context.junit4.SpringRunner;

@RunWith(SpringRunner.class)
@SpringBootTest
public class ActiveScheduleDaoImplFailureTest {

  @Autowired private ActiveScheduleDao activeScheduleDao;

  @MockitoSpyBean private DataSource dataSource;

  @Autowired TestDataDbUtil testDataDbUtil;

  @Before
  public void before() throws SQLException, InterruptedException {
    Mockito.reset(dataSource);
    Mockito.when(dataSource.getConnection()).thenThrow(new SQLException("test exception"));
  }

  @Test
  public void testFindActiveSchedule_throw_DatabaseValidationException() throws SQLException {
    try {
      activeScheduleDao.find(1L);
      fail("Should fail");
    } catch (DatabaseValidationException dve) {
      assertThat(dve.getMessage(), is("Find failed"));
    }
  }

  @Test
  public void testCreateActiveSchedule_throw_DatabaseValidationException() {
    try {
      activeScheduleDao.create(new ActiveScheduleEntity());

      fail("Should fail");
    } catch (DatabaseValidationException dve) {
      assertThat(dve.getMessage(), is("Create failed"));
    }
  }

  @Test
  public void testDeleteActiveSchedule_throw_DatabaseValidationException()
      throws SQLException, InterruptedException {

    try {
      activeScheduleDao.delete(null, null);

      fail("Should fail");
    } catch (DatabaseValidationException dve) {
      assertThat(dve.getMessage(), is("Delete failed"));
    }
  }

  @Test
  public void testFindActiveScheduleByAppId_throw_DatabaseValidationException()
      throws SQLException {
    String appId = "appId_1";
    try {
      activeScheduleDao.findByAppId(appId);
      fail("Should fail");
    } catch (DatabaseValidationException dve) {
      assertThat(
          dve.getMessage(), is("Select active schedules by Application Id:" + appId + " failed"));
    }
  }
}
