package org.cloudfoundry.autoscaler.scheduler.service;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNull;
import static org.junit.Assert.fail;

import java.util.List;

import javax.transaction.Transactional;

import org.cloudfoundry.autoscaler.scheduler.dao.ScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
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
	private ScheduleDao scheduleDao;

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
		Mockito.reset(scheduleDao);
		removeAllRecoredsFromDatabase();
	}

	@Transactional
	public void removeAllRecoredsFromDatabase() {
		List<String> appIds = TestDataSetupHelper.getAllGeneratedAppIds();
		for (String appId : appIds) {
			for (ScheduleEntity entity : scheduleDao.findAllSchedulesByAppId(appId)) {
				scheduleDao.delete(entity);
			}
		}
	}

	@Test
	@Transactional
	public void testGetAllSchedules_with_no_schedules() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		ApplicationScalingSchedules scalingSchedules = scheduleManager.getAllSchedules(appId);
		assertNull(scalingSchedules);

	}

	@Test
	@Transactional
	public void testCreateAndGetAllSchedule() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		assertCreateAndFindAllSchedules(appId, 4);

	}

	@Test
	@Rollback
	public void testCreateSchedule_throw_DatabaseValidationException() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		ApplicationScalingSchedules schedules = TestDataSetupHelper.generateSpecificDateSchedules(appId, 1);

		ScheduleEntity entity = schedules.getSpecific_date().get(0);
		entity.setEndDate(null);

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
	public void testFindAllSchedule_throw_DatabaseValidationException() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		Mockito.when(scheduleDao.findAllSchedulesByAppId(Mockito.anyString()))
				.thenThrow(new DatabaseValidationException());

		try {
			scheduleManager.getAllSchedules(appId);
			fail("Expected failure case.");
		} catch (SchedulerInternalException e) {
			String message = messageBundleResourceHelper.lookupMessage("database.error.get.failed",
					"app_id=" + appId);

			for (String errorMessage : validationErrorResult.getAllErrorMessages()) {
				assertEquals(message, errorMessage);
			}
		}
	}

	private void assertCreateAndFindAllSchedules(String appId, int noOfSpecificDateSchedules) {
		createScheduleNotThrowAnyException(appId, noOfSpecificDateSchedules);

		List<ScheduleEntity> foundSpecificSchedules = scheduleManager.getAllSchedules(appId).getSpecific_date();
		assertSpecificSchedulesFoundEquals(noOfSpecificDateSchedules, foundSpecificSchedules);

	}

	private void assertSpecificSchedulesFoundEquals(int expectedScheduleTobeFound,
			List<ScheduleEntity> foundSchedules) {
		assertEquals(expectedScheduleTobeFound, foundSchedules.size());
	}

	private void createScheduleNotThrowAnyException(String appId, int noOfSpecificDateSchedules) {
		ApplicationScalingSchedules schedules = TestDataSetupHelper.generateSpecificDateSchedules(appId,
				noOfSpecificDateSchedules);
		scheduleManager.createSchedules(schedules);
	}
}
