package org.cloudfoundry.autoscaler.scheduler.dao;

import static org.hamcrest.Matchers.is;
import static org.hamcrest.Matchers.nullValue;
import static org.junit.Assert.assertThat;

import java.util.List;

import javax.persistence.EntityManager;
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

@RunWith(SpringRunner.class)
@SpringBootTest
@Transactional
public class ActiveScheduleDaoImplTest extends TestConfiguration {

	@Autowired
	private EntityManager entityManager;

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
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		Long scheduleId = 1L;
		ActiveScheduleEntity activeScheduleEntity = TestDataSetupHelper.generateActiveScheduleEntity(appId, scheduleId,
				JobActionEnum.START);
		activeScheduleDao.create(activeScheduleEntity);

		appId = TestDataSetupHelper.generateAppIds(1)[0];
		scheduleId = 2L;
		activeScheduleEntity = TestDataSetupHelper.generateActiveScheduleEntity(appId, scheduleId, JobActionEnum.START);
		activeScheduleDao.create(activeScheduleEntity);
	}

	@Test
	public void testFindActiveSchedule_with_incorrect_Id() {
		assertThat("It should be null", activeScheduleDao.find(3L), nullValue());
	}

	@Test
	public void testCreateAndFindActiveSchedule() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		Long scheduleId = 3L;
		ActiveScheduleEntity activeScheduleEntity = TestDataSetupHelper.generateActiveScheduleEntity(appId, scheduleId,
				JobActionEnum.START);
		activeScheduleDao.create(activeScheduleEntity);

		ActiveScheduleEntity foundActiveScheduleEntity = activeScheduleDao.find(activeScheduleEntity.getId());

		assertThat("It should be 3", getActiveSchedulesCount(), is(3L));
		assertThat("Both active schedules should be equal", activeScheduleEntity, is(foundActiveScheduleEntity));
	}

	@Test
	public void testDeleteActiveSchedule() {
		Long activeScheduleId = 2L;
		int deletedActiveSchedules = activeScheduleDao.delete(activeScheduleId);

		assertThat("It should be 1", deletedActiveSchedules, is(1));
		assertThat("It should be 1", getActiveSchedulesCount(), is(1L));
		assertThat("It should be null", activeScheduleDao.find(activeScheduleId), nullValue());
	}

	@Test
	public void testDeleteActiveSchedule_with_nullValue() {

		int number = activeScheduleDao.delete(null);

		assertThat("It should be 0", number, is(0));
		assertThat("It should be 2", getActiveSchedulesCount(), is(2L));
	}

	@Test
	public void testDeleteActiveSchedule_with_invalidId() {

		int number = activeScheduleDao.delete(7L);

		assertThat("It should be 0", number, is(0));
		assertThat("It should be 2", getActiveSchedulesCount(), is(2L));
	}

	@Test
	public void testFindAllActiveSchedules() {
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

		List<ActiveScheduleEntity> activeSchedules = activeScheduleDao.findAllActiveSchedulesByAppId(appId);

		assertThat("It should have count of 3 active schedules", activeSchedules.size(), is(3));

		for (ActiveScheduleEntity activeScheduleEntityFound : activeSchedules) {
			assertThat("It should have expected app id", activeScheduleEntityFound.getAppId(), is(appId));
		}
	}

	private long getActiveSchedulesCount() {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
		return jdbcTemplate.queryForObject("SELECT COUNT(1) FROM app_scaling_active_schedule", Long.class);
	}
}
