package org.cloudfoundry.autoscaler.scheduler.service;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.fail;

import java.util.List;

import javax.transaction.Transactional;

import org.cloudfoundry.autoscaler.scheduler.dao.RecurringScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.dao.SpecificDateScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationScalingSchedules;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.DatabaseValidationException;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.SchedulerInternalException;
import org.cloudfoundry.autoscaler.scheduler.util.error.ValidationErrorResult;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.Mockito;
import org.quartz.Scheduler;
import org.quartz.SchedulerException;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.annotation.DirtiesContext;
import org.springframework.test.annotation.DirtiesContext.ClassMode;
import org.springframework.test.annotation.Rollback;
import org.springframework.test.context.ActiveProfiles;
import org.springframework.test.context.junit4.SpringRunner;

/**
 * 
 */
@ActiveProfiles("ScheduleDaoMock")
@RunWith(SpringRunner.class)
@SpringBootTest
@DirtiesContext(classMode = ClassMode.BEFORE_EACH_TEST_METHOD)
public class ScheduleManagerTest {

	@Autowired
	private ScheduleManager scheduleManager;

	@Autowired
	private SpecificDateScheduleDao specificDateScheduleDao;

	@Autowired
	private RecurringScheduleDao recurringScheduleDao;

	@Autowired
	private Scheduler scheduler;

	@Autowired
	MessageBundleResourceHelper messageBundleResourceHelper;

	@Autowired
	private ValidationErrorResult validationErrorResult;

	@Before
	@Transactional
	public void init() throws SchedulerException {
		// Clear previous schedules.
		scheduler.clear();
		Mockito.reset(specificDateScheduleDao);
		removeAllRecordsFromDatabase();
	}

	@Transactional
	public void removeAllRecordsFromDatabase() {
		List<String> appIds = TestDataSetupHelper.getAllGeneratedAppIds();
		for (String appId : appIds) {
			for (SpecificDateScheduleEntity entity : specificDateScheduleDao
					.findAllSpecificDateSchedulesByAppId(appId)) {
				specificDateScheduleDao.delete(entity);
			}
		}
	}

	@Test
	@Transactional
	public void testGetAllSchedules_with_no_schedules() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		ApplicationScalingSchedules scalingSchedules = scheduleManager.getAllSchedules(appId);
		assertFalse(scalingSchedules.hasSchedules());

	}

	@Test
	@Transactional
	public void testCreateAndGetAllSchedules() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		// Create 4 specific date schedules and no recurring schedules then get them.
		assertCreateAndFindAllSchedules(appId, 4, 0);

		appId = TestDataSetupHelper.generateAppIds(1)[0];
		// Create no specific date schedules and 4 recurring schedules then get them.
		assertCreateAndFindAllSchedules(appId, 0, 4);

		appId = TestDataSetupHelper.generateAppIds(1)[0];
		// Create 4 specific date schedules and 4 recurring schedules then get them.
		assertCreateAndFindAllSchedules(appId, 4, 4);
	}

	@Test
	@Rollback
	public void testCreateSpecificDateSchedule_throw_DatabaseValidationException() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		ApplicationScalingSchedules schedules = TestDataSetupHelper.generateSchedules(appId, 1, 0);

		SpecificDateScheduleEntity entity = schedules.getSpecific_date().get(0);
		entity.setEndDateTime(null);

		try {
			scheduleManager.createSchedules(schedules);
			fail("Expected failure case.");
		} catch (SchedulerInternalException e) {
			String message = messageBundleResourceHelper.lookupMessage("database.error.create.failed",
					"app_id=" + entity.getAppId());

			for (String errorMessage : validationErrorResult.getAllErrorMessages()) {
				assertEquals(message, errorMessage);
			}
		}
	}

	@Test
	@Rollback
	public void testCreateRecurringSchedule_throw_DatabaseValidationException() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		ApplicationScalingSchedules schedules = TestDataSetupHelper.generateSchedules(appId, 0, 1);

		RecurringScheduleEntity entity = schedules.getRecurring_schedule().get(0);
		entity.setStartTime(null);

		try {
			scheduleManager.createSchedules(schedules);
			fail("Expected failure case.");
		} catch (SchedulerInternalException e) {
			String message = messageBundleResourceHelper.lookupMessage("database.error.create.failed",
					"app_id=" + entity.getAppId());

			for (String errorMessage : validationErrorResult.getAllErrorMessages()) {
				assertEquals(message, errorMessage);
			}
		}
	}

	@Test
	@Rollback
	public void testFindAllSpecificDateSchedule_throw_DatabaseValidationException() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		Mockito.when(specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(Mockito.anyString()))
				.thenThrow(new DatabaseValidationException());

		try {
			scheduleManager.getAllSchedules(appId);
			fail("Expected failure case.");
		} catch (SchedulerInternalException e) {
			String message = messageBundleResourceHelper.lookupMessage("database.error.get.failed", "app_id=" + appId);

			for (String errorMessage : validationErrorResult.getAllErrorMessages()) {
				assertEquals(message, errorMessage);
			}
		}
	}

	@Test
	@Rollback
	public void testFindAllRecurringSchedule_throw_DatabaseValidationException() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		Mockito.when(recurringScheduleDao.findAllRecurringSchedulesByAppId(Mockito.anyString()))
				.thenThrow(new DatabaseValidationException());

		try {
			scheduleManager.getAllSchedules(appId);
			fail("Expected failure case.");
		} catch (SchedulerInternalException e) {
			String message = messageBundleResourceHelper.lookupMessage("database.error.get.failed", "app_id=" + appId);

			for (String errorMessage : validationErrorResult.getAllErrorMessages()) {
				assertEquals(message, errorMessage);
			}
		}
	}

	private void assertCreateAndFindAllSchedules(String appId, int noOfSpecificDateSchedules,
			int noOfRecurringSchedules) {
		createScheduleNotThrowAnyException(appId, noOfSpecificDateSchedules, noOfRecurringSchedules);

		ApplicationScalingSchedules schedules = scheduleManager.getAllSchedules(appId);

		assertSpecificDateSchedulesFoundEquals(noOfSpecificDateSchedules, schedules.getSpecific_date());
		assertRecurringSchedulesFoundEquals(noOfRecurringSchedules, schedules.getRecurring_schedule());

	}

	private void assertRecurringSchedulesFoundEquals(int noOfRecurringSchedules,
			List<RecurringScheduleEntity> recurringSchedules) {
		if (recurringSchedules != null) {
			assertEquals(noOfRecurringSchedules, recurringSchedules.size());
		} else {
			assertEquals(noOfRecurringSchedules, 0);
		}

	}

	private void assertSpecificDateSchedulesFoundEquals(int expectedScheduleTobeFound,
			List<SpecificDateScheduleEntity> foundSchedules) {
		if (foundSchedules != null) {
			assertEquals(expectedScheduleTobeFound, foundSchedules.size());
		} else {
			assertEquals(expectedScheduleTobeFound, 0);
		}
	}

	private void createScheduleNotThrowAnyException(String appId, int noOfSpecificDateSchedules,
			int noOfRecurringSchedules) {
		ApplicationScalingSchedules schedules = TestDataSetupHelper.generateSchedules(appId, noOfSpecificDateSchedules,
				noOfRecurringSchedules);
		scheduleManager.createSchedules(schedules);
	}
}
