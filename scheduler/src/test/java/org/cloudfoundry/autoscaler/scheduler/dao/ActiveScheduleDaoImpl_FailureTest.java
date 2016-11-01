package org.cloudfoundry.autoscaler.scheduler.dao;

import static org.hamcrest.Matchers.is;
import static org.junit.Assert.assertThat;
import static org.junit.Assert.fail;

import java.sql.SQLException;

import javax.sql.DataSource;

import org.apache.logging.log4j.Level;
import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.core.LoggerContext;
import org.apache.logging.log4j.core.config.Configuration;
import org.apache.logging.log4j.core.config.LoggerConfig;
import org.cloudfoundry.autoscaler.scheduler.entity.ActiveScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataCleanupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.DatabaseValidationException;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.Mockito;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.test.context.ActiveProfiles;
import org.springframework.test.context.junit4.SpringRunner;

@ActiveProfiles("DataSourceMock")
@RunWith(SpringRunner.class)
@SpringBootTest
public class ActiveScheduleDaoImpl_FailureTest {


	@Autowired
	private ActiveScheduleDao activeScheduleDao;

	@Autowired
	private DataSource dataSource;

	@Autowired
	TestDataCleanupHelper testDataCleanupHelper;

	@Before
	public void before() throws SQLException, InterruptedException {
		setLogLevel(Level.INFO);

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
	public void testDeleteActiveSchedule_throw_DatabaseValidationException() throws SQLException, InterruptedException {

		try {
			activeScheduleDao.delete(null);

			fail("Should fail");
		} catch (DatabaseValidationException dve) {
			assertThat(dve.getMessage(), is("Delete failed"));
		}
	}

	private void setLogLevel(Level level) {
		LoggerContext ctx = (LoggerContext) LogManager.getContext(false);
		Configuration config = ctx.getConfiguration();

		LoggerConfig loggerConfig = config.getLoggerConfig(LogManager.ROOT_LOGGER_NAME);
		loggerConfig.removeAppender("MockAppender");

		loggerConfig.setLevel(level);
		ctx.updateLoggers();
	}

	private long getActiveSchedulesCount() {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
		return jdbcTemplate.queryForObject("SELECT COUNT(1) FROM app_scaling_active_schedule", Long.class);
	}
}
