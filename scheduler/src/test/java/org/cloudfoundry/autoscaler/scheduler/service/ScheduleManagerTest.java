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

		List<ScheduleEntity> allSpecificDateSchedules = scheduleManager.getAllSchedules(appId)
				.getSpecific_date();

		assertEquals(0, allSpecificDateSchedules.size());

	}

	@Test
	@Transactional
	public void testGetSchedule_02() {
		// Expected one schedule
	
		ApplicationScalingSchedules schedules = TestDataSetupHelper.generateSpecificDateSchedules(appId, 1);

		scheduleManager.createSchedules(schedules);

		List<ScheduleEntity> allSpecificDateSchedules = scheduleManager.getAllSchedules(appId).getSpecific_date();

		assertEquals(1, allSpecificDateSchedules.size());

	}
	
	@Test
	@Transactional
	public void testGetSchedule_03() {
		// Expected multiple schedules
		
		int noOfSpecificDateSchedules = 4;
		
		ApplicationScalingSchedules schedules = TestDataSetupHelper.generateSpecificDateSchedules(appId,
				noOfSpecificDateSchedules);

		scheduleManager.createSchedules(schedules);

		List<ScheduleEntity> allSpecificDateSchedules = scheduleManager.getAllSchedules(appId).getSpecific_date();

		assertEquals(noOfSpecificDateSchedules, allSpecificDateSchedules.size());

	}

	@Test
	@Transactional
	public void testCreateSchedules_04() throws Exception {
		// Create one schedule
		ApplicationScalingSchedules schedules = TestDataSetupHelper.generateSpecificDateSchedules(appId, 1);

		scheduleManager.createSchedules(schedules);

		List<ScheduleEntity> allSpecificDateSchedules = scheduleManager.getAllSchedules(appId).getSpecific_date();

		assertEquals(1, allSpecificDateSchedules.size());
	}

	@Test
	@Transactional
	public void testCreateSchedules_05() throws Exception {
		// Create multiple schedules
		
		int noOfSpecificDateSchedules = 4;

		ApplicationScalingSchedules schedules = TestDataSetupHelper.generateSpecificDateSchedules(appId,
				noOfSpecificDateSchedules);

		scheduleManager.createSchedules(schedules);

		List<ScheduleEntity> allSpecificDateSchedules = scheduleManager.getAllSchedules(appId).getSpecific_date();

		assertEquals(noOfSpecificDateSchedules, allSpecificDateSchedules.size());
	}

}
