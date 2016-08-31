package org.cloudfoundry.autoscaler.scheduler.service;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNotNull;
import static org.junit.Assert.assertTrue;

import java.util.Calendar;
import java.util.Date;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.TimeZone;

import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.JobActionEnum;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleTypeEnum;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
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
import org.quartz.Trigger;
import org.quartz.TriggerKey;
import org.quartz.impl.matchers.GroupMatcher;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.annotation.DirtiesContext;
import org.springframework.test.annotation.DirtiesContext.ClassMode;
import org.springframework.test.context.ActiveProfiles;
import org.springframework.test.context.junit4.SpringRunner;

/**
 * 
 *
 */
@ActiveProfiles("SchedulerMock")
@RunWith(SpringRunner.class)
@SpringBootTest
@DirtiesContext(classMode = ClassMode.BEFORE_EACH_TEST_METHOD)
public class ScheduleJobManagerTest {

	@Autowired
	private Scheduler scheduler;

	@Autowired
	private ScheduleJobManager scalingJobManager;

	@Autowired
	private ValidationErrorResult validationErrorResult;

	@Autowired
	private MessageBundleResourceHelper messageBundleResourceHelper;

	private Long scheduleIdx = 0L;

	@Before
	public void initializer() throws SchedulerException {
		// Clear previous schedules.
		scheduler.clear();
		Mockito.reset(scheduler);
	}

	@Test
	public void testCreateAndFindSimpleJobs() throws Exception {
		int schedulesToSetup = 4;
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		List<SpecificDateScheduleEntity> specificDateScheduleEntities = TestDataSetupHelper
				.generateSpecificDateScheduleEntities(appId, schedulesToSetup);

		createSimpleJob(specificDateScheduleEntities);

		// The expected number of jobs would be twice the number of schedules(
		// One job for start and one for end)
		int expectedJobsToBeCreated = 2 * schedulesToSetup;

		Map<JobKey, JobDetail> scheduleJobKeyDetailMap = getSchedulerJobs(
				ScheduleTypeEnum.SPECIFIC_DATE.getScheduleIdentifier());
		// Check expected jobs created
		assertEquals(expectedJobsToBeCreated, scheduleJobKeyDetailMap.size());

		for (ScheduleEntity scheduleEntity : specificDateScheduleEntities) {
			assertCreatedJobs(scheduleJobKeyDetailMap, scheduleEntity, ScheduleTypeEnum.SPECIFIC_DATE);
		}
	}

	@Test
	public void testCreateAndFindCronJobs() throws Exception {
		int noOfDOMRecurringSchedules = 2;
		int noOfDOWRecurringSchedules = 2;
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		List<RecurringScheduleEntity> recurringScheduleEntities = TestDataSetupHelper
				.generateRecurringScheduleEntities(appId, noOfDOMRecurringSchedules, noOfDOWRecurringSchedules);

		createCronJob(recurringScheduleEntities);

		// The expected number of jobs would be twice the number of schedules
		int expectedJobsToBeCreated = 2 * (noOfDOMRecurringSchedules + noOfDOMRecurringSchedules);
		ScheduleTypeEnum scheduleTypeEnum = ScheduleTypeEnum.RECURRING;
		Map<JobKey, JobDetail> scheduleJobKeyDetailMap = getSchedulerJobs(
				ScheduleTypeEnum.RECURRING.getScheduleIdentifier());

		// Check expected jobs fire day
		assertEquals(expectedJobsToBeCreated, scheduleJobKeyDetailMap.size());

		for (ScheduleEntity scheduleEntity : recurringScheduleEntities) {
			assertCreatedJobs(scheduleJobKeyDetailMap, scheduleEntity, scheduleTypeEnum);
		}
	}

	@Test
	public void testCreateCronJob_with_dayOfWeek() throws Exception {
		checkNextFireJobTheDayOfWeek(Calendar.MONDAY, TimeZone.getTimeZone("America/Phoenix"));
		checkNextFireJobTheDayOfWeek(Calendar.FRIDAY, TimeZone.getTimeZone("GMT"));
		checkNextFireJobTheDayOfWeek(Calendar.SUNDAY, TimeZone.getTimeZone("Pacific/Kiritimati"));
	}

	@Test
	public void testCreateCronJob_with_dayOfMonth() throws Exception {
		checkNextFireJobTheDayOfMonth(9, TimeZone.getTimeZone("Chile/Continental"), "2100-10-10");
		checkNextFireJobTheDayOfMonth(27, TimeZone.getTimeZone("GMT"), null);
		checkNextFireJobTheDayOfMonth(14, TimeZone.getTimeZone("Europe/Amsterdam"), null);
	}

	@Test
	public void testCreateSimpleJob_with_throw_SchedulerException_at_Quartz() throws SchedulerException {

		// Set mock object for Quartz.
		Mockito.doThrow(SchedulerException.class).when(scheduler).scheduleJob(Mockito.anyObject(), Mockito.anyObject());
		int noOfSpecificDateSchedulesToSetUp = 1;
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		List<SpecificDateScheduleEntity> specificDateScheduleEntities = TestDataSetupHelper
				.generateSpecificDateScheduleEntities(appId, noOfSpecificDateSchedulesToSetUp);
		createSimpleJob(specificDateScheduleEntities);

		assertTrue("This test should have an Error.", validationErrorResult.hasErrors());

		List<String> errors = validationErrorResult.getAllErrorMessages();
		assertEquals(1, errors.size());

		String errorMessage = messageBundleResourceHelper.lookupMessage("scheduler.error.create.failed",
				"app_id=" + appId, null);

		assertEquals(errorMessage, errors.get(0));
	}

	@Test
	public void testCreateCronJob_with_throw_SchedulerException_at_Quartz() throws SchedulerException {

		// Set mock object for Quartz.
		Mockito.doThrow(SchedulerException.class).when(scheduler).scheduleJob(Mockito.anyObject(), Mockito.anyObject());
		int noOfDOMRecurringSchedules = 1;
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		List<RecurringScheduleEntity> recurringScheduleEntities = TestDataSetupHelper
				.generateRecurringScheduleEntities(appId, noOfDOMRecurringSchedules, 0);
		Long index = 0L;
		for (RecurringScheduleEntity scheduleEntity : recurringScheduleEntities) {
			scheduleEntity.setId(++index);
			scalingJobManager.createCronJob(scheduleEntity);
		}

		assertTrue("This test should have an Error.", validationErrorResult.hasErrors());
		List<String> errors = validationErrorResult.getAllErrorMessages();
		assertEquals(1, errors.size());

		String errorMessage = messageBundleResourceHelper.lookupMessage("scheduler.error.create.failed",
				"app_id=" + appId, null);

		assertEquals(errorMessage, errors.get(0));
	}

	@Test
	public void testDeleteSimpleJobs() throws Exception {
		int noOfSpecificDateSchedulesToSetUp = 2;
		int expectedJobsToBeCreated = 4; // 2 jobs per schedule, one for start
											// and one for end
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		List<SpecificDateScheduleEntity> specificDateScheduleEntities = TestDataSetupHelper
				.generateSpecificDateScheduleEntities(appId, noOfSpecificDateSchedulesToSetUp);

		createSimpleJob(specificDateScheduleEntities);
		Map<JobKey, JobDetail> scheduleJobKeyDetailMap = getSchedulerJobs(
				ScheduleTypeEnum.SPECIFIC_DATE.getScheduleIdentifier());

		// Check expected jobs created
		assertEquals(expectedJobsToBeCreated, scheduleJobKeyDetailMap.size());

		// Delete the simple jobs
		for (ScheduleEntity scheduleEntity : specificDateScheduleEntities) {
			scalingJobManager.deleteJob(scheduleEntity.getAppId(), scheduleEntity.getId(),
					ScheduleTypeEnum.SPECIFIC_DATE);
		}

		scheduleJobKeyDetailMap = getSchedulerJobs(ScheduleTypeEnum.SPECIFIC_DATE.getScheduleIdentifier());
		// Check the jobs, the expected job count is 0.
		assertEquals(0, scheduleJobKeyDetailMap.size());

	}

	@Test
	public void testDeleteCronJobs() throws Exception {
		int noOfDOMSchedules = 1;
		int noOfDOWSchedules = 1;
		int expectedJobsToBeCreated = 4; // 2 jobs per schedule, one for start
											// and one for end
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		List<RecurringScheduleEntity> recurringScheduleEntities = TestDataSetupHelper
				.generateRecurringScheduleEntities(appId, noOfDOMSchedules, noOfDOWSchedules);

		createCronJob(recurringScheduleEntities);
		Map<JobKey, JobDetail> scheduleJobKeyDetailMap = getSchedulerJobs(
				ScheduleTypeEnum.RECURRING.getScheduleIdentifier());

		// Check expected jobs created
		assertEquals(expectedJobsToBeCreated, scheduleJobKeyDetailMap.size());

		// Delete the cron jobs
		for (ScheduleEntity scheduleEntity : recurringScheduleEntities) {
			scalingJobManager.deleteJob(scheduleEntity.getAppId(), scheduleEntity.getId(), ScheduleTypeEnum.RECURRING);
		}

		scheduleJobKeyDetailMap = getSchedulerJobs(ScheduleTypeEnum.RECURRING.getScheduleIdentifier());
		// Check the jobs, the expected job count is 0.
		assertEquals(0, scheduleJobKeyDetailMap.size());

	}

	@Test
	public void testDeleteSimpleJob_with_throw_SchedulerException_at_Quartz() throws SchedulerException {

		// Set mock object for Quartz.
		Mockito.doThrow(SchedulerException.class).when(scheduler).deleteJob(Mockito.anyObject());
		int noOfSpecificDateSchedulesToSetUp = 1;
		int expectedJobsToBeCreated = 2; // 2 jobs per schedule, one for start
											// and one for end

		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		List<SpecificDateScheduleEntity> specificDateScheduleEntities = TestDataSetupHelper
				.generateSpecificDateScheduleEntities(appId, noOfSpecificDateSchedulesToSetUp);
		createSimpleJob(specificDateScheduleEntities);

		Map<JobKey, JobDetail> scheduleJobKeyDetailMap = getSchedulerJobs(
				ScheduleTypeEnum.SPECIFIC_DATE.getScheduleIdentifier());

		// Check expected jobs created
		assertEquals(expectedJobsToBeCreated, scheduleJobKeyDetailMap.size());

		// Delete the simple jobs
		for (ScheduleEntity scheduleEntity : specificDateScheduleEntities) {
			scalingJobManager.deleteJob(scheduleEntity.getAppId(), scheduleEntity.getId(),
					ScheduleTypeEnum.SPECIFIC_DATE);
		}

		assertTrue("This test should have an Error.", validationErrorResult.hasErrors());

		List<String> errors = validationErrorResult.getAllErrorMessages();
		assertEquals(1, errors.size());

		String errorMessage = messageBundleResourceHelper.lookupMessage("scheduler.error.delete.failed",
				"app_id=" + appId, null);

		assertEquals(errorMessage, errors.get(0));

		scheduleJobKeyDetailMap = getSchedulerJobs(ScheduleTypeEnum.SPECIFIC_DATE.getScheduleIdentifier());

		// Check jobs still exist
		assertEquals(expectedJobsToBeCreated, scheduleJobKeyDetailMap.size());

	}

	@Test
	public void testDeleteCronJob_with_throw_SchedulerException_at_Quartz() throws SchedulerException {

		// Set mock object for Quartz.
		Mockito.doThrow(SchedulerException.class).when(scheduler).deleteJob(Mockito.anyObject());
		int noOfDOMRecurringSchedules = 1;
		int expectedJobsToBeCreated = 2; // 2 jobs per schedule, one for start
											// and one for end

		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		List<RecurringScheduleEntity> recurringScheduleEntities = TestDataSetupHelper
				.generateRecurringScheduleEntities(appId, noOfDOMRecurringSchedules, 0);

		createCronJob(recurringScheduleEntities);
		Map<JobKey, JobDetail> scheduleJobKeyDetailMap = getSchedulerJobs(
				ScheduleTypeEnum.RECURRING.getScheduleIdentifier());

		// Check expected jobs created
		assertEquals(expectedJobsToBeCreated, scheduleJobKeyDetailMap.size());

		// Delete the cron jobs
		for (ScheduleEntity scheduleEntity : recurringScheduleEntities) {
			scalingJobManager.deleteJob(scheduleEntity.getAppId(), scheduleEntity.getId(), ScheduleTypeEnum.RECURRING);
		}

		assertTrue("This test should have an Error.", validationErrorResult.hasErrors());
		List<String> errors = validationErrorResult.getAllErrorMessages();
		assertEquals(1, errors.size());

		String errorMessage = messageBundleResourceHelper.lookupMessage("scheduler.error.delete.failed",
				"app_id=" + appId, null);

		assertEquals(errorMessage, errors.get(0));

		scheduleJobKeyDetailMap = getSchedulerJobs(ScheduleTypeEnum.RECURRING.getScheduleIdentifier());

		// Check jobs
		assertEquals(expectedJobsToBeCreated, scheduleJobKeyDetailMap.size());

	}

	private void createSimpleJob(List<SpecificDateScheduleEntity> specificDateScheduleEntities) {
		for (SpecificDateScheduleEntity scheduleEntity : specificDateScheduleEntities) {
			Long scheduleId = ++scheduleIdx;
			scheduleEntity.setId(scheduleId);
			scalingJobManager.createSimpleJob(scheduleEntity);
		}
	}

	private void createCronJob(List<RecurringScheduleEntity> recurringScheduleEntities) {
		for (RecurringScheduleEntity scheduleEntity : recurringScheduleEntities) {
			Long scheduleId = ++scheduleIdx;
			scheduleEntity.setId(scheduleId);
			scalingJobManager.createCronJob(scheduleEntity);
		}
	}

	private Map<JobKey, JobDetail> getSchedulerJobs(String groupName) throws SchedulerException {
		Map<JobKey, JobDetail> scheduleJobKeyDetailMap = new HashMap<>();

		for (JobKey jobKey : scheduler.getJobKeys(GroupMatcher.jobGroupEquals(groupName))) {
			scheduleJobKeyDetailMap.put(jobKey, scheduler.getJobDetail(jobKey));

		}

		return scheduleJobKeyDetailMap;
	}

	private Trigger getSchedulerTrigger(Long scheduleIdx, JobActionEnum jobActionEnum, String groupName)
			throws SchedulerException {
		String name = scheduleIdx + jobActionEnum.getJobIdSuffix();
		Trigger trigger = null;

		for (TriggerKey triggerKey : scheduler.getTriggerKeys(GroupMatcher.triggerGroupEquals(groupName))) {
			if (triggerKey.getName().startsWith(name)) {
				trigger = scheduler.getTrigger(triggerKey);
			}
		}

		return trigger;
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

	private void checkNextFireJobTheDayOfWeek(int day, TimeZone timeZone) throws SchedulerException {
		int[] dayOfWeek = { TestDataSetupHelper.convertIntToCalendarDayOfWeek(day) };

		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		List<RecurringScheduleEntity> recurringScheduleEntities = TestDataSetupHelper
				.generateRecurringScheduleEntities(appId, 0, 1);

		RecurringScheduleEntity entity = recurringScheduleEntities.get(0);
		entity.setTimeZone(timeZone.getID());
		entity.setDaysOfWeek(dayOfWeek);

		createCronJob(recurringScheduleEntities);

		// Check expected jobs fire day
		Trigger startTrigger = getSchedulerTrigger(this.scheduleIdx, JobActionEnum.START,
				ScheduleTypeEnum.RECURRING.getScheduleIdentifier());
		Trigger endTrigger = getSchedulerTrigger(this.scheduleIdx, JobActionEnum.END,
				ScheduleTypeEnum.RECURRING.getScheduleIdentifier());

		assertNextFireJobDayOfWeek(day, entity.getStartTime(), timeZone, startTrigger);
		assertNextFireJobDayOfWeek(day, entity.getEndTime(), timeZone, endTrigger);
	}

	private void assertNextFireJobDayOfWeek(int expectedDay, Date scheduleTime, TimeZone timeZone,
			Trigger actualTrigger) {
		Calendar expectedTime = Calendar.getInstance();
		expectedTime.setTime(scheduleTime);

		Calendar actualCal = Calendar.getInstance();
		actualCal.setTime(actualTrigger.getNextFireTime());
		actualCal.setTimeZone(timeZone);

		assertEquals(expectedDay, actualCal.get(Calendar.DAY_OF_WEEK));
		assertEquals(expectedTime.get(Calendar.HOUR_OF_DAY), actualCal.get(Calendar.HOUR_OF_DAY));
		assertEquals(expectedTime.get(Calendar.MINUTE), actualCal.get(Calendar.MINUTE));
	}

	private void checkNextFireJobTheDayOfMonth(int day, TimeZone timeZone, String startDate) throws Exception {
		int[] dayOfMonth = { day };

		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		List<RecurringScheduleEntity> recurringScheduleEntities = TestDataSetupHelper
				.generateRecurringScheduleEntities(appId, 1, 0);

		RecurringScheduleEntity entity = recurringScheduleEntities.get(0);
		entity.setTimeZone(timeZone.getID());
		entity.setDaysOfMonth(dayOfMonth);
		entity.setStartDate(TestDataSetupHelper.getDate(startDate));

		createCronJob(recurringScheduleEntities);

		// Check expected jobs fire day
		Calendar now = Calendar.getInstance();
		now.setTimeZone(timeZone);
		Calendar calStartDate = Calendar.getInstance();
		if (entity.getStartDate() != null) {
			calStartDate.setTime(entity.getStartDate());
		}

		if (calStartDate.get(Calendar.DAY_OF_MONTH) > day) {
			calStartDate.add(Calendar.MONTH, 1);
		}

		Trigger startTrigger = getSchedulerTrigger(this.scheduleIdx, JobActionEnum.START,
				ScheduleTypeEnum.RECURRING.getScheduleIdentifier());
		Trigger endTrigger = getSchedulerTrigger(this.scheduleIdx, JobActionEnum.END,
				ScheduleTypeEnum.RECURRING.getScheduleIdentifier());

		assertNextFireDateTime(day, timeZone, entity.getStartTime(), calStartDate, startTrigger);
		assertNextFireDateTime(day, timeZone, entity.getEndTime(), calStartDate, endTrigger);

	}

	private void assertNextFireDateTime(int expectedDay, TimeZone timeZone, Date scheduleTime,
			Calendar expectedStartDate, Trigger trigger) {
		Calendar expectedTime = Calendar.getInstance();
		expectedTime.setTime(scheduleTime);

		Calendar actualCal = Calendar.getInstance();
		actualCal.setTime(trigger.getNextFireTime());
		actualCal.setTimeZone(timeZone);

		assertEquals(expectedStartDate.get(Calendar.YEAR), actualCal.get(Calendar.YEAR));
		assertEquals(expectedStartDate.get(Calendar.MONTH), actualCal.get(Calendar.MONTH));
		assertEquals(expectedDay, actualCal.get(Calendar.DAY_OF_MONTH));
		assertEquals(expectedTime.get(Calendar.HOUR_OF_DAY), actualCal.get(Calendar.HOUR_OF_DAY));
		assertEquals(expectedTime.get(Calendar.MINUTE), actualCal.get(Calendar.MINUTE));
	}
}
