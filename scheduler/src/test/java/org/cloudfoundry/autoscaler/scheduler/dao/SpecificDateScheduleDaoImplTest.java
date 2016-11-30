package org.cloudfoundry.autoscaler.scheduler.dao;

import static org.hamcrest.Matchers.is;
import static org.junit.Assert.assertThat;
import static org.junit.Assert.fail;

import java.util.Date;
import java.util.List;

import javax.sql.DataSource;
import javax.transaction.Transactional;

import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
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
public class SpecificDateScheduleDaoImplTest extends TestConfiguration {

	@Autowired
	private SpecificDateScheduleDao specificDateScheduleDao;

	@Autowired
	private TestDataCleanupHelper testDataCleanupHelper;

	@Autowired
	private DataSource dataSource;

	@Before
	public void before() {
		// Remove All ActiveSchedules
		testDataCleanupHelper.cleanupData();

		// Add fake test records.
		String appId = "appId1";
		insertSpecificDateSchedule(appId, "GMT", 1, 5, 2, 7, 0, new Date(), new Date());

		appId = "appId3";
		insertSpecificDateSchedule(appId, "GMT", 1, 5, 2, 7, 0, new Date(), new Date());
	}

	@Test
	public void testFindAllSpecificDateSchedulesByAppId_with_invalidAppId() {
		String appId = "invalid_appId";

		List<SpecificDateScheduleEntity> specificDateScheduleEntities = specificDateScheduleDao
				.findAllSpecificDateSchedulesByAppId(appId);

		assertThat("It should have empty list", specificDateScheduleEntities.isEmpty(), is(true));
	}

	@Test
	public void testFindAllSpecificDateSchedulesByAppId() {
		String appId = "appId1";

		List<SpecificDateScheduleEntity> foundEntityList = specificDateScheduleDao
				.findAllSpecificDateSchedulesByAppId(appId);

		assertThat("It should have one specific date schedule", foundEntityList.size(), is(1));
		assertThat("The appId should be equal", foundEntityList.get(0).getAppId(), is(appId));

	}

	@Test
	public void testCreateSpecificDateSchedule() {
		String appId = "appId2";
		SpecificDateScheduleEntity specificDateScheduleEntity = TestDataSetupHelper
				.generateSpecificDateScheduleEntities(appId, 1).get(0);

		assertThat("It should have no specific date schedule", getRecurringSchedulesCountByAppId(appId), is(0L));

		SpecificDateScheduleEntity savedEntity = specificDateScheduleDao.create(specificDateScheduleEntity);

		Long currentSequenceSchedulerId = testDataCleanupHelper.getCurrentSequenceSchedulerId();
		specificDateScheduleEntity.setId(currentSequenceSchedulerId);

		assertThat("It should have one specific date schedule", getRecurringSchedulesCountByAppId(appId), is(1L));
		assertThat("Both recurring schedules should be equal", savedEntity, is(specificDateScheduleEntity));
	}

	@Test
	public void testDeleteSchedule() {
		String appId = "appId1";

		assertThat("It should have three records", getRecurringSchedulesCountByAppId(appId), is(1L));

		SpecificDateScheduleEntity specificDateScheduleEntity = specificDateScheduleDao
				.findAllSpecificDateSchedulesByAppId(appId).get(0);
		specificDateScheduleDao.delete(specificDateScheduleEntity);

		assertThat("It should have three records", getRecurringSchedulesCountByAppId(appId), is(0L));
	}

	@Test
	public void testDeleteSchedule_with_invalidAppId() {
		SpecificDateScheduleEntity specificDateScheduleEntity = new SpecificDateScheduleEntity();
		specificDateScheduleEntity.setAppId("invalid_appId");

		assertThat("It should have three records", getRecurringSchedulesCount(), is(2L));

		specificDateScheduleDao.delete(specificDateScheduleEntity);

		assertThat("It should have three records", getRecurringSchedulesCount(), is(2L));
	}

	@Test
	public void testFindSchedulesByAppId_throw_Exception() {
		try {
			specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(null);
			fail("Should fail");
		} catch (DatabaseValidationException dve) {
			assertThat(dve.getMessage(), is("Find All specific date schedules failed"));
		}
	}

	@Test
	public void testCreateSchedule_throw_Exception() {
		try {
			specificDateScheduleDao.create(null);
			fail("Should fail");
		} catch (DatabaseValidationException dve) {
			assertThat(dve.getMessage(), is("Create failed"));
		}
	}

	@Test
	public void testDeleteSchedule_throw_Exception() {
		try {
			specificDateScheduleDao.delete(null);
			fail("Should fail");
		} catch (DatabaseValidationException dve) {
			assertThat(dve.getMessage(), is("Delete failed"));
		}
	}

	private void insertSpecificDateSchedule(String appId, String timezone, int defaultInstanceMinCount,
			int defaultInstanceMaxCount, int instanceMinCount, int instanceMaxCount, int initialMinInstanceCount,
			Date startDateTime, Date endDateTime) {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);

		Long scheduleId = getNextValFromSequence();
		Object[] objects = new Object[] { scheduleId, appId, timezone, defaultInstanceMinCount, defaultInstanceMaxCount,
				instanceMinCount, instanceMaxCount, initialMinInstanceCount, startDateTime, endDateTime };

		jdbcTemplate.update("INSERT INTO app_scaling_specific_date_schedule "
				+ "( schedule_id, app_id, timezone, default_instance_min_count, default_instance_max_count, instance_min_count, instance_max_count, initial_min_instance_count, start_date_time, end_date_time) "
				+ "VALUES ( ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", objects);
	}

	private Long getNextValFromSequence() {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
		return jdbcTemplate.queryForObject("SELECT nextval('schedule_id_sequence');", Long.class);
	}

	private long getRecurringSchedulesCount() {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
		return jdbcTemplate.queryForObject("SELECT COUNT(1) FROM app_scaling_specific_date_schedule", Long.class);
	}

	private long getRecurringSchedulesCountByAppId(String appId) {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
		return jdbcTemplate.queryForObject("SELECT COUNT(1) FROM app_scaling_specific_date_schedule WHERE app_id=?",
				new Object[] { appId }, Long.class);
	}
}
