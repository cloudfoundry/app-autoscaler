package org.cloudfoundry.autoscaler.scheduler.service;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertNotNull;
import static org.junit.Assert.fail;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

import javax.transaction.Transactional;

import org.cloudfoundry.autoscaler.scheduler.dao.RecurringScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.dao.SpecificDateScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationScalingSchedules;
import org.cloudfoundry.autoscaler.scheduler.util.JobActionEnum;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleTypeEnum;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.DatabaseValidationException;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.SchedulerInternalException;
import org.cloudfoundry.autoscaler.scheduler.util.error.ValidationErrorResult;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.Mockito;
import org.quartz.JobDataMap;
import org.quartz.JobDetail;
import org.quartz.JobKey;
import org.quartz.Scheduler;
import org.quartz.SchedulerException;
import org.quartz.impl.matchers.GroupMatcher;
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
	private MessageBundleResourceHelper messageBundleResourceHelper;

	@Autowired
	private ValidationErrorResult validationErrorResult;

	@Before
	@Transactional
	public void init() throws SchedulerException {
		// Clear previous schedules.
		scheduler.clear();
		Mockito.reset(specificDateScheduleDao);
		Mockito.reset(recurringScheduleDao);
		removeData();
	}

	@Transactional
	public void removeData() {
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
	public void testCreateAndGetAllSchedules() throws Exception {
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
	@Transactional
	public void testCreateAndGetSchedues_Timezone() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		int noOfSpecificDateSchedules = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper.generateSchedulesWithEntitiesOnly(appId,
				noOfSpecificDateSchedules, 0, 0);
		scheduleManager.createSchedules(schedules);
		// Create 1 specific date schedule.
		createScheduleNotThrowAnyException(appId, 4, 0);

	}

	@Test
	@Rollback
	public void testCreateSpecificDateSchedule_throw_DatabaseValidationException() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		ApplicationScalingSchedules schedules = TestDataSetupHelper.generateSchedulesWithEntitiesOnly(appId, 1, 0, 0);

		SpecificDateScheduleEntity entity = schedules.getSpecific_date().get(0);
		entity.setEndDateTime(null);

		assertDatabaseExceptionOnCreate(appId, schedules);
	}

	@Test
	@Rollback
	public void testCreateRecurringSchedule_throw_DatabaseValidationException() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];

		ApplicationScalingSchedules schedules = TestDataSetupHelper.generateSchedulesWithEntitiesOnly(appId, 0, 1, 0);

		RecurringScheduleEntity entity = schedules.getRecurring_schedule().get(0);
		entity.setStartTime(null);

		assertDatabaseExceptionOnCreate(appId, schedules);
	}

	@Test
	@Transactional
	public void testFindAllSpecificDateSchedule_throw_DatabaseValidationException() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		assertDatabaseExceptionOnFind(appId, ScheduleTypeEnum.SPECIFIC_DATE);
	}

	@Test
	@Transactional
	public void testFindAllRecurringSchedule_throw_DatabaseValidationException() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		assertDatabaseExceptionOnFind(appId, ScheduleTypeEnum.RECURRING);
	}

	@Test
	@Transactional
	public void testDeleteSchedules() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];

		// Create 4 specific date schedules and 4 recurring schedules then get them.
		createScheduleNotThrowAnyException(appId, 4, 4);

		// Get schedules and assert to check schedules are created
		ApplicationScalingSchedules schedules = scheduleManager.getAllSchedules(appId);
		assertSpecificDateSchedulesFoundEquals(4, schedules.getSpecific_date());
		assertRecurringSchedulesFoundEquals(4, schedules.getRecurring_schedule());

		scheduleManager.deleteSchedules(appId);

		// Get schedules and assert to check there are no schedules
		schedules = scheduleManager.getAllSchedules(appId);
		assertSpecificDateSchedulesFoundEquals(0, schedules.getSpecific_date());
		assertRecurringSchedulesFoundEquals(0, schedules.getRecurring_schedule());
	}

	@Test
	@Rollback
	public void testDeleteSpecificDateSchedules_throw_DatabaseValidationException() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];

		// Create 4 specific date schedules.
		createScheduleNotThrowAnyException(appId, 4, 0);

		assertDatabaseExceptionOnDelete(appId, ScheduleTypeEnum.SPECIFIC_DATE);
	}

	@Test
	@Rollback
	public void testDeleteRecurringSchedules_throw_DatabaseValidationException() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];

		// Create 4 recurring schedules.
		createScheduleNotThrowAnyException(appId, 0, 4);

		assertDatabaseExceptionOnDelete(appId, ScheduleTypeEnum.RECURRING);
	}

	private void createScheduleNotThrowAnyException(String appId, int noOfSpecificDateSchedules,
			int noOfRecurringSchedules) {

		int noOfDOMRecurringSchedules = noOfRecurringSchedules / 2;
		int noOfDOWRecurringSchedules = noOfRecurringSchedules - noOfDOMRecurringSchedules;
		ApplicationScalingSchedules schedules = TestDataSetupHelper.generateSchedulesWithEntitiesOnly(appId, noOfSpecificDateSchedules,
				noOfDOMRecurringSchedules, noOfDOWRecurringSchedules);
		scheduleManager.createSchedules(schedules);
	}

	private void assertCreateAndFindAllSchedules(String appId, int noOfSpecificDateSchedules,
			int noOfRecurringSchedules) throws Exception {
		createScheduleNotThrowAnyException(appId, noOfSpecificDateSchedules, noOfRecurringSchedules);

		ApplicationScalingSchedules schedules = scheduleManager.getAllSchedules(appId);

		assertSpecificDateSchedulesFoundEquals(noOfSpecificDateSchedules, schedules.getSpecific_date());
		assertRecurringSchedulesFoundEquals(noOfRecurringSchedules, schedules.getRecurring_schedule());

		if (schedules.getSpecific_date() != null) {
			int expectedSpecificDateJobsToBeCreated = noOfSpecificDateSchedules * 2;
			Map<JobKey, JobDetail> scheduleJobKeyDetailMap = getSchedulerJobs(appId,
					ScheduleTypeEnum.SPECIFIC_DATE.getScheduleIdentifier());

			// Check expected jobs created
			assertEquals(expectedSpecificDateJobsToBeCreated, scheduleJobKeyDetailMap.size());

			for (ScheduleEntity scheduleEntity : schedules.getSpecific_date()) {
				assertCreatedJobs(scheduleJobKeyDetailMap, scheduleEntity, ScheduleTypeEnum.SPECIFIC_DATE);
			}
		}

		if (schedules.getRecurring_schedule() != null) {
			// Check expected jobs created
			Map<JobKey, JobDetail> recurringScheduleJobKeyDetailMap = getSchedulerJobs(appId,
					ScheduleTypeEnum.RECURRING.getScheduleIdentifier());

			int expectedRecurringJobsToBeCreated = noOfRecurringSchedules * 2;
			assertEquals(expectedRecurringJobsToBeCreated, recurringScheduleJobKeyDetailMap.size());

			for (ScheduleEntity scheduleEntity : schedules.getRecurring_schedule()) {
				assertCreatedJobs(recurringScheduleJobKeyDetailMap, scheduleEntity, ScheduleTypeEnum.RECURRING);
			}
		}
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

	private void assertDatabaseExceptionOnFind(String appId, ScheduleTypeEnum scheduleTypeEnum) {
		if (scheduleTypeEnum == ScheduleTypeEnum.SPECIFIC_DATE) {
			Mockito.when(specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(Mockito.anyString()))
					.thenThrow(new DatabaseValidationException());
		} else {
			Mockito.when(recurringScheduleDao.findAllRecurringSchedulesByAppId(Mockito.anyString()))
					.thenThrow(new DatabaseValidationException());
		}

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

	private void assertDatabaseExceptionOnCreate(String appId, ApplicationScalingSchedules schedules) {
		try {
			scheduleManager.createSchedules(schedules);
			fail("Expected failure case.");
		} catch (SchedulerInternalException e) {
			String message = messageBundleResourceHelper.lookupMessage("database.error.create.failed",
					"app_id=" + appId);

			for (String errorMessage : validationErrorResult.getAllErrorMessages()) {
				assertEquals(message, errorMessage);
			}
		}
	}

	private void assertDatabaseExceptionOnDelete(String appId, ScheduleTypeEnum scheduleTypeEnum) {
		if (scheduleTypeEnum == ScheduleTypeEnum.SPECIFIC_DATE) {
			Mockito.doThrow(DatabaseValidationException.class).when(specificDateScheduleDao)
					.delete(Mockito.anyObject());
		} else {
			Mockito.doThrow(DatabaseValidationException.class).when(recurringScheduleDao).delete(Mockito.anyObject());
		}

		try {
			scheduleManager.deleteSchedules(appId);
			fail("Expected failure case.");
		} catch (SchedulerInternalException e) {
			String message = messageBundleResourceHelper.lookupMessage("database.error.delete.failed",
					"app_id=" + appId);

			for (String errorMessage : validationErrorResult.getAllErrorMessages()) {
				assertEquals(message, errorMessage);
			}
		}
	}

	private Map<JobKey, JobDetail> getSchedulerJobs(String appId, String groupName) throws SchedulerException {
		Map<JobKey, JobDetail> scheduleJobKeyDetailMap = new HashMap<>();

		for (JobKey jobKey : scheduler.getJobKeys(GroupMatcher.jobGroupEquals(groupName))) {
			JobDetail detail = scheduler.getJobDetail(jobKey);
			if (detail.getJobDataMap().get("appId").equals(appId)) {
				scheduleJobKeyDetailMap.put(jobKey, detail);
			}
		}

		return scheduleJobKeyDetailMap;
	}

	private void assertCreatedJobs(Map<JobKey, JobDetail> scheduleIdJobDetailMap, ScheduleEntity scheduleEntity,
			ScheduleTypeEnum scheduleType) throws SchedulerException {
		String appId = scheduleEntity.getAppId();
		Long scheduleId = scheduleEntity.getId();

		JobKey startJobKey = ScheduleJobHelper.generateJobKey(scheduleId, JobActionEnum.START, scheduleType);
		JobKey endJobKey = ScheduleJobHelper.generateJobKey(scheduleId, JobActionEnum.END, scheduleType);

		int instMinCount = scheduleEntity.getInstanceMinCount();
		int instMaxCount = scheduleEntity.getInstanceMaxCount();

		JobDetail jobDetail = scheduleIdJobDetailMap.get(startJobKey);
		assertJobDetails(appId, scheduleId, instMinCount, instMaxCount, JobActionEnum.START, jobDetail);

		instMinCount = scheduleEntity.getDefaultInstanceMinCount();
		instMaxCount = scheduleEntity.getDefaultInstanceMaxCount();
		jobDetail = scheduleIdJobDetailMap.get(endJobKey);
		assertJobDetails(appId, scheduleId, instMinCount, instMaxCount, JobActionEnum.END, jobDetail);
	}

	private void assertJobDetails(String expectedAppId, Long expectedScheduleId, int expectedInstanceMinCount,
			int expectedInstanceMaxCount, JobActionEnum expectedJobAction, JobDetail expectedJobDetail) {
		assertNotNull("Expected existing jobDetail", expectedJobDetail);
		JobDataMap map = expectedJobDetail.getJobDataMap();
		assertEquals(expectedAppId, map.get("appId"));
		assertEquals(expectedScheduleId, map.get("scheduleId"));
		assertEquals(expectedJobAction, map.get("scalingAction"));
		assertEquals(expectedInstanceMinCount, map.get("instanceMinCount"));
		assertEquals(expectedInstanceMaxCount, map.get("instanceMaxCount"));
	}

}
