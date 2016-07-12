package org.cloudfoundry.autoscaler.scheduler.service;

import static org.junit.Assert.assertEquals;

import java.util.List;

import javax.transaction.Transactional;

import org.cloudfoundry.autoscaler.scheduler.dao.ScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationScalingSchedules;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.ValidationErrorResult;
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

	@Autowired
	private ValidationErrorResult validationErrorResult;

	private String appId = TestDataSetupHelper.getAppId_1();

	@Before
	@Transactional
	public void init() throws SchedulerException {
		// Clear previous schedules.
		scheduler.clear();
		validationErrorResult.initEmpty();
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
	public void testGetSchedule_01() {
		// Expected no schedule

		List<ScheduleEntity> allSpecificDateSchedules = scheduleManager.getAllSchedules(appId).getSpecific_date();
		assertEquals(0, allSpecificDateSchedules.size());

	}

	@Test
	@Transactional
	public void testGetSchedule_03() {
		// Expected multiple schedules
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

	private void assertSpecificSchedulesFoundEquals(int expectedScheduleTobeFound, List<ScheduleEntity> foundSchedules) {
		assertEquals(expectedScheduleTobeFound, foundSchedules.size());
	}

	private void createScheduleNotThrowAnyException(int noOfSpecificDateSchedules) {
		ApplicationScalingSchedules schedules = TestDataSetupHelper.generateSpecificDateSchedules(appId,
				noOfSpecificDateSchedules);
		scheduleManager.createSchedules(schedules);
	}
}
