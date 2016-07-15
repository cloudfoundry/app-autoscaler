package org.cloudfoundry.autoscaler.scheduler.service;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNull;

import java.util.List;

import javax.transaction.Transactional;

import org.cloudfoundry.autoscaler.scheduler.dao.ScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationScalingSchedules;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.junit.After;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.quartz.Scheduler;
import org.quartz.SchedulerException;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.annotation.DirtiesContext;
import org.springframework.test.context.junit4.SpringRunner;

/**
 * 
 */
@RunWith(SpringRunner.class)
@SpringBootTest
@DirtiesContext(classMode = DirtiesContext.ClassMode.AFTER_EACH_TEST_METHOD)
public class ScheduleManagerTest {

	@Autowired
	private ScheduleManager scheduleManager;

	@Autowired
	private ScheduleDao scheduleDao;

	@Autowired
	private Scheduler scheduler;

	private String appId = TestDataSetupHelper.generateAppIds(1)[0];

	@Before
	@Transactional
	public void init() throws SchedulerException {
		// Clear previous schedules.
		scheduler.clear();
	}

	@After
	@Transactional
	public void afterTest() {
		for (ScheduleEntity entity : scheduleDao.findAllSchedulesByAppId(appId)) {
			scheduleDao.delete(entity);
		}

	}

	@Test
	@Transactional
	public void testGetAllSchedules_with_no_schedules() {
		ApplicationScalingSchedules scalingSchedules = scheduleManager.getAllSchedules(appId);
		assertNull(scalingSchedules);

	}

	@Test
	@Transactional
	public void testCreateAndGetAllSchedule() {
		assertCreateAndFindAllSchedules(1);
		assertCreateAndFindAllSchedules(4);

	}

	private void assertCreateAndFindAllSchedules(int noOfSpecificDateSchedules) {
		createScheduleNotThrowAnyException(noOfSpecificDateSchedules);

		List<ScheduleEntity> foundSpecificSchedules = scheduleManager.getAllSchedules(appId).getSpecific_date();
		assertSpecificSchedulesFoundEquals(noOfSpecificDateSchedules, foundSpecificSchedules);

		// reset all records for next test.
		afterTest();
	}

	private void assertSpecificSchedulesFoundEquals(int expectedScheduleTobeFound,
			List<ScheduleEntity> foundSchedules) {
		assertEquals(expectedScheduleTobeFound, foundSchedules.size());
	}

	private void createScheduleNotThrowAnyException(int noOfSpecificDateSchedules) {
		ApplicationScalingSchedules schedules = TestDataSetupHelper.generateSpecificDateSchedules(appId,
				noOfSpecificDateSchedules);
		scheduleManager.createSchedules(schedules);
	}
}
