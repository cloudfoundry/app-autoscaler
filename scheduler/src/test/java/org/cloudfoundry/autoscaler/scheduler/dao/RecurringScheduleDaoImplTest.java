package org.cloudfoundry.autoscaler.scheduler.dao;

import static org.hamcrest.Matchers.is;
import static org.junit.Assert.assertThat;
import static org.junit.Assert.fail;

import java.sql.SQLException;
import java.sql.Time;
import java.util.Date;
import java.util.List;

import javax.sql.DataSource;
import javax.transaction.Transactional;

import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.TestConfiguration;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataCleanupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.DatabaseValidationException;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.test.context.junit4.SpringRunner;

@RunWith(SpringRunner.class)
@SpringBootTest
@Transactional
public class RecurringScheduleDaoImplTest extends TestConfiguration {
	@Autowired
	private RecurringScheduleDao recurringScheduleDao;

	@Autowired
	private TestDataCleanupHelper testDataCleanupHelper;

	@Autowired
	private DataSource dataSource;

	@Before
	public void before() throws SQLException {
		// Remove All ActiveSchedules
		testDataCleanupHelper.cleanupData();

		// Add fake test records.
		String appId = "appId1";
		insertRecurringSchedule(appId, "GMT", 1, 5, 2, 7, 0, Time.valueOf("01:00:00"), Time.valueOf("23:00:00"), null,
				null, new int[] { 1, 3, 5 }, null);

		appId = "appId3";
		insertRecurringSchedule(appId, "GMT", 1, 5, 2, 7, 0, Time.valueOf("01:00:00"), Time.valueOf("23:00:00"),
				new Date(), null, null, new int[] { 1, 5, 10, 20 });
	}

	@Test
	public void testFindAllSchedules_with_invalidAppId() {
		String appId = "invalid_appId";

		List<RecurringScheduleEntity> recurringScheduleEntities = recurringScheduleDao
				.findAllRecurringSchedulesByAppId(appId);

		assertThat("It should be empty list", recurringScheduleEntities.isEmpty(), is(true));
	}

	@Test
	public void testFindAllRecurringSchedulesByAppId() {
		String appId = "appId3";

		List<RecurringScheduleEntity> foundEntityList = recurringScheduleDao.findAllRecurringSchedulesByAppId(appId);

		assertThat("It should have one record", foundEntityList.size(), is(1));
		assertThat("The appId should be equal", foundEntityList.get(0).getAppId(), is(appId));
	}

	@Test
	public void testCreateRecurringSchedule() {
		String appId = "appId2";

		RecurringScheduleEntity recurringScheduleEntity = TestDataSetupHelper
				.generateRecurringScheduleEntities(appId, 1, 0).get(0);

		assertThat("It should no recurring schedule", getRecurringSchedulesCountByAppId(appId), is(0L));

		RecurringScheduleEntity savedEntity = recurringScheduleDao.create(recurringScheduleEntity);

		Long currentSequenceSchedulerId = testDataCleanupHelper.getCurrentSequenceSchedulerId();
		recurringScheduleEntity.setId(currentSequenceSchedulerId);

		assertThat("It should have one recurring schedule", getRecurringSchedulesCountByAppId(appId), is(1L));
		assertThat("Both recurring schedules should be equal", savedEntity, is(recurringScheduleEntity));
	}

	@Test
	public void testDeleteSchedule() {
		String appId = "appId1";
		assertThat("It should have one recurring schedule", getRecurringSchedulesCountByAppId(appId), is(1L));
		RecurringScheduleEntity recurringScheduleEntity = recurringScheduleDao.findAllRecurringSchedulesByAppId(appId)
				.get(0);
		recurringScheduleDao.delete(recurringScheduleEntity);

		assertThat("It should have no recurring schedule", getRecurringSchedulesCountByAppId(appId), is(0L));
	}

	@Test
	public void testDeleteSchedule_with_invalidAppId() {
		String appId = "invalid_appId";
		RecurringScheduleEntity recurringScheduleEntity = new RecurringScheduleEntity();
		recurringScheduleEntity.setAppId(appId);

		assertThat("There are two recurring schedules", getRecurringSchedulesCount(), is(2L));

		recurringScheduleDao.delete(recurringScheduleEntity);

		assertThat("There are two recurring schedules", getRecurringSchedulesCount(), is(2L));
	}

	@Test
	public void testFindSchedulesByAppId_throw_Exception() {
		try {
			recurringScheduleDao.findAllRecurringSchedulesByAppId(null);
			fail("Should fail");
		} catch (DatabaseValidationException dve) {
			assertThat(dve.getMessage(), is("Find All recurring schedules failed"));
		}
	}

	@Test
	public void testCreateSchedule_throw_Exception() {
		try {
			recurringScheduleDao.create(null);
			fail("Should fail");
		} catch (DatabaseValidationException dve) {
			assertThat(dve.getMessage(), is("Create failed"));
		}
	}

	@Test
	public void testDeleteSchedule_throw_Exception() {
		try {
			recurringScheduleDao.delete(null);
			fail("Should fail");
		} catch (DatabaseValidationException dve) {
			assertThat(dve.getMessage(), is("Delete failed"));
		}
	}

	private void insertRecurringSchedule(String appId, String timezone, int defaultInstanceMinCount,
			int defaultInstanceMaxCount, int instanceMinCount, int instanceMaxCount, int initialMinInstanceCount,
			Time startTime, Time endTime, Date startDate, Date endDate, int[] daysOfWeek, int[] daysOfMonth) {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);

		Long scheduleId = getNextValFromSequence();
		Object[] objects = new Object[] { scheduleId, appId, timezone, defaultInstanceMinCount, defaultInstanceMaxCount,
				instanceMinCount, instanceMaxCount, initialMinInstanceCount, startTime, endTime, startDate, endDate,
				convertArrayToBits(daysOfWeek), convertArrayToBits(daysOfMonth) };

		jdbcTemplate.update("INSERT INTO app_scaling_recurring_schedule "
				+ "( schedule_id, app_id, timezone, default_instance_min_count, default_instance_max_count, instance_min_count, instance_max_count, initial_min_instance_count, start_time, end_time, start_date, end_date, days_of_week, days_of_month) "
				+ "VALUES ( ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", objects);
	}

	private Long getNextValFromSequence() {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
		return jdbcTemplate.queryForObject("SELECT nextval('schedule_id_sequence');", Long.class);
	}

	private long getRecurringSchedulesCount() {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
		return jdbcTemplate.queryForObject("SELECT COUNT(1) FROM app_scaling_recurring_schedule", Long.class);
	}

	private long getRecurringSchedulesCountByAppId(String appId) {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
		return jdbcTemplate.queryForObject("SELECT COUNT(1) FROM app_scaling_recurring_schedule WHERE app_id=?",
				new Object[] { appId }, Long.class);
	}

	private int convertArrayToBits(int[] values) {
		int bits = 0;

		if (values == null) {
			bits = 0;
		} else {
			for (int value : values) {
				bits |= 1 << (value - 1);
			}

		}
		return bits;
	}

}
