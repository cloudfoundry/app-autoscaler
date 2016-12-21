package org.cloudfoundry.autoscaler.scheduler.dao;

import static org.hamcrest.Matchers.is;
import static org.hamcrest.Matchers.nullValue;
import static org.junit.Assert.assertThat;

import javax.sql.DataSource;
import javax.transaction.Transactional;

import org.cloudfoundry.autoscaler.scheduler.entity.ActiveScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.JobActionEnum;
import org.cloudfoundry.autoscaler.scheduler.util.TestConfiguration;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataCleanupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.test.context.junit4.SpringRunner;

import java.util.List;

@RunWith(SpringRunner.class)
@SpringBootTest
@Transactional
public class ActiveScheduleDaoImplTest extends TestConfiguration {

	@Autowired
	private ActiveScheduleDao activeScheduleDao;

	@Autowired
	private DataSource dataSource;

	@Autowired
	private TestDataCleanupHelper testDataCleanupHelper;

	@Before
	public void before() {
		// Remove All ActiveSchedules
		testDataCleanupHelper.cleanupData();

		// Add fake test records.
		String appId = "appId_1";
		Long scheduleId = 1L;
		Long startJobIdentifier = 1L;
		insertActiveSchedule(appId, scheduleId, 1, 5, 0, startJobIdentifier);

		appId = "appId_2";
		scheduleId = 2L;
		startJobIdentifier = 2L;
		insertActiveSchedule(appId, scheduleId, 2, 7, 3, startJobIdentifier);
	}

	@Test
	public void testFindActiveSchedule_with_invalidId() {
		ActiveScheduleEntity activeScheduleEntity = activeScheduleDao.find(3L);
		assertThat("It should be null", activeScheduleEntity, nullValue());
	}

	@Test
	public void testCreateAndFindActiveSchedule() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		Long scheduleId = 3L;
		ActiveScheduleEntity activeScheduleEntity = TestDataSetupHelper.generateActiveScheduleEntity(appId, scheduleId,
				JobActionEnum.START);

		assertThat("It should have no active schedule", getActiveSchedulesCountByScheduleId(scheduleId), is(0L));

		activeScheduleDao.create(activeScheduleEntity);

		assertThat("It should have one active schedule", getActiveSchedulesCountByScheduleId(scheduleId), is(1L));

		ActiveScheduleEntity foundActiveScheduleEntity = activeScheduleDao.find(activeScheduleEntity.getId());

		assertThat("Both active schedules should be equal", activeScheduleEntity, is(foundActiveScheduleEntity));
	}

	@Test
	public void testDeleteActiveSchedule() {
		Long activeScheduleId = 2L;
		Long startJobIdentifier = 2L;

		assertThat("It should have one active schedule", getActiveSchedulesCountByScheduleId(activeScheduleId), is(1L));

		int deletedActiveSchedules = activeScheduleDao.delete(activeScheduleId, startJobIdentifier);

		assertThat("It should be 1", deletedActiveSchedules, is(1));
		assertThat("It should have no active schedule", getActiveSchedulesCountByScheduleId(activeScheduleId), is(0L));
	}

	@Test
	public void testDeleteActiveSchedule_with_nullValue() {

		assertThat("It should be 2", getActiveSchedulesCount(), is(2L));

		int number = activeScheduleDao.delete(null, null);

		assertThat("It should be 0", number, is(0));
		assertThat("It should be 2", getActiveSchedulesCount(), is(2L));
	}

	@Test
	public void testDeleteActiveSchedule_with_invalidId() {

		assertThat("It should be 2", getActiveSchedulesCount(), is(2L));

		int deletedActiveSchedules = activeScheduleDao.delete(7L, 3L);

		assertThat("It should be 0", deletedActiveSchedules, is(0));
		assertThat("It should be 2", getActiveSchedulesCount(), is(2L));
	}

	@Test
	public void testDeleteAllActiveSchedulesByAppId() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		Long scheduleId = 3L;
		ActiveScheduleEntity activeScheduleEntity = TestDataSetupHelper.generateActiveScheduleEntity(appId, scheduleId,
				JobActionEnum.START);
		activeScheduleDao.create(activeScheduleEntity);

		scheduleId = 4L;
		activeScheduleEntity = TestDataSetupHelper.generateActiveScheduleEntity(appId, scheduleId, JobActionEnum.START);
		activeScheduleDao.create(activeScheduleEntity);

		scheduleId = 5L;
		activeScheduleEntity = TestDataSetupHelper.generateActiveScheduleEntity(appId, scheduleId, JobActionEnum.START);
		activeScheduleDao.create(activeScheduleEntity);

		assertThat("It should have 3 active schedules", getActiveSchedulesCountByAppId(appId), is(3L));

		activeScheduleDao.deleteActiveSchedulesByAppId(appId);

		assertThat("It should have no active schedules", getActiveSchedulesCountByAppId(appId), is(0L));

	}

	@Test
	public void testFindActiveScheduleByAppId() {
		String appId = "appId_1";
		List<ActiveScheduleEntity> activeScheduleEntities = activeScheduleDao.findByAppId(appId);
		assertThat("It should have one active schedule", activeScheduleEntities.size(), is(1));
	}

	@Test
	public void testFindActiveScheduleByAppId_with_invalidAppId() {
		String appId = "invalid_appId";
		List<ActiveScheduleEntity> activeScheduleEntities = activeScheduleDao.findByAppId(appId);
		assertThat("It should have no active schedule", activeScheduleEntities.size(), is(0));
	}

	private void insertActiveSchedule(String appId, Long scheduleId, int instanceMinCount, int instanceMaxCount,
			int initialMinInstanceCount, Long startJobIdentifier) {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);

		Object[] objects = new Object[] { scheduleId, appId, startJobIdentifier, instanceMinCount, instanceMaxCount,
				initialMinInstanceCount };

		jdbcTemplate.update("INSERT INTO app_scaling_active_schedule "
				+ "(id, app_id, start_job_identifier, instance_min_count, instance_max_count, initial_min_instance_count) "
				+ "VALUES (?, ?, ?, ?, ?, ?)", objects);
	}

	private long getActiveSchedulesCount() {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
		return jdbcTemplate.queryForObject("SELECT COUNT(1) FROM app_scaling_active_schedule", Long.class);
	}

	private long getActiveSchedulesCountByAppId(String appId) {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
		return jdbcTemplate.queryForObject("SELECT COUNT(1) FROM app_scaling_active_schedule WHERE app_id=?",
				new Object[] { appId }, Long.class);
	}

	private long getActiveSchedulesCountByScheduleId(Long scheduleId) {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
		return jdbcTemplate.queryForObject("SELECT COUNT(1) FROM app_scaling_active_schedule WHERE id=?",
				new Object[] { scheduleId }, Long.class);
	}
}
