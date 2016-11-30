package org.cloudfoundry.autoscaler.scheduler.service;

import static org.hamcrest.Matchers.is;
import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNotNull;
import static org.junit.Assert.assertThat;
import static org.junit.Assert.assertTrue;
import static org.junit.Assert.fail;
import static org.mockito.Matchers.eq;

import java.sql.Time;
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
import org.cloudfoundry.autoscaler.scheduler.util.RecurringScheduleEntitiesBuilder;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleTypeEnum;
import org.cloudfoundry.autoscaler.scheduler.util.SpecificDateScheduleEntitiesBuilder;
import org.cloudfoundry.autoscaler.scheduler.util.TestConfiguration;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataCleanupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.ValidationErrorResult;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.ArgumentCaptor;
import org.mockito.Mockito;
import org.quartz.CronTrigger;
import org.quartz.JobDataMap;
import org.quartz.JobDetail;
import org.quartz.JobKey;
import org.quartz.Scheduler;
import org.quartz.SchedulerException;
import org.quartz.Trigger;
import org.quartz.TriggerKey;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.test.context.junit4.SpringRunner;

@RunWith(SpringRunner.class)
@SpringBootTest
public class ScheduleJobManagerTest extends TestConfiguration {

	@MockBean
	private Scheduler scheduler;

	@Autowired
	private ScheduleJobManager scheduleJobManager;

	@Autowired
	private ValidationErrorResult validationErrorResult;

	@Autowired
	private MessageBundleResourceHelper messageBundleResourceHelper;

	@Autowired
	private TestDataCleanupHelper testDataCleanupHelper;

	@Before
	public void before() throws SchedulerException {
		testDataCleanupHelper.cleanupData();

		Mockito.reset(scheduler);
	}

	@Test
	public void testCreateSimpleJobs_with_GMT_timeZone() throws Exception {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String timeZone = TimeZone.getTimeZone("GMT").getID();
		Date startDateTime = new Date();
		Date endDateTime = new Date();

		Date expectedStartDateTime = new Date(startDateTime.getTime() + TimeZone.getDefault().getRawOffset()
				- TimeZone.getTimeZone(timeZone).getRawOffset());
		Date expectedEndDateTime = new Date(endDateTime.getTime() + TimeZone.getDefault().getRawOffset()
				- TimeZone.getTimeZone(timeZone).getRawOffset());

		SpecificDateScheduleEntity specificDateScheduleEntity = new SpecificDateScheduleEntitiesBuilder(1)
				.setAppid(appId).setTimeZone(timeZone).setScheduleId().setStartDateTime(0, startDateTime)
				.setEndDateTime(0, endDateTime).setDefaultInstanceMinCount(1).setDefaultInstanceMaxCount(5).build()
				.get(0);

		Long id = specificDateScheduleEntity.getId();
		ScheduleTypeEnum scheduleTypeEnum = ScheduleTypeEnum.SPECIFIC_DATE;
		JobKey startJobKey = ScheduleJobHelper.generateJobKey(id, JobActionEnum.START, scheduleTypeEnum);
		JobKey endJobKey = ScheduleJobHelper.generateJobKey(id, JobActionEnum.END, scheduleTypeEnum);

		TriggerKey startTriggerKey = ScheduleJobHelper.generateTriggerKey(id, JobActionEnum.START, scheduleTypeEnum);
		TriggerKey endTriggerKey = ScheduleJobHelper.generateTriggerKey(id, JobActionEnum.END, scheduleTypeEnum);

		scheduleJobManager.createSimpleJob(specificDateScheduleEntity);

		assertThat("Never call it", validationErrorResult.hasErrors(), is(false));

		ArgumentCaptor<JobDetail> jobDetailArgumentCaptor = ArgumentCaptor.forClass(JobDetail.class);
		ArgumentCaptor<Trigger> triggerArgumentCaptor = ArgumentCaptor.forClass(Trigger.class);

		Mockito.verify(scheduler, Mockito.times(2)).scheduleJob(jobDetailArgumentCaptor.capture(),
				triggerArgumentCaptor.capture());

		Map<JobKey, JobDetail> scheduleJobKeyDetailMap = new HashMap<>();
		for (JobDetail jobDetail : jobDetailArgumentCaptor.getAllValues()) {
			scheduleJobKeyDetailMap.put(jobDetail.getKey(), jobDetail);
		}
		assertCreatedJobs(scheduleJobKeyDetailMap, specificDateScheduleEntity, scheduleTypeEnum);

		for (Trigger trigger : triggerArgumentCaptor.getAllValues()) {
			if (trigger.getKey().equals(startTriggerKey)) {
				assertThat(trigger.getJobKey(), is(startJobKey));
				assertThat(trigger.getStartTime(), is(expectedStartDateTime));
			} else if (trigger.getKey().equals(endTriggerKey)) {
				assertThat(trigger.getJobKey(), is(endJobKey));
				assertThat(trigger.getStartTime(), is(expectedEndDateTime));
			} else {
				fail("Invalid trigger key :" + trigger.getKey());
			}
		}
	}

	@Test
	public void testCreateCronJob_with_dayOfWeek_GMT() throws Exception {
		String timeZone = "GMT";

		String startTime = "22:10:00";
		String endTime = "23:20:00";
		int[] dayOfWeek = { 2, 4, 6 };

		String expectedCronExpressionForStartJob = "00 10 22 ? * TUE,THU,SAT *";
		String expectedCronExpressionForEndJob = "00 20 23 ? * TUE,THU,SAT *";

		RecurringScheduleEntity recurringScheduleEntity = createRecurringScheduleWithDaysOfWeek(timeZone, startTime,
				endTime, dayOfWeek);

		Long id = recurringScheduleEntity.getId();
		ScheduleTypeEnum scheduleType = ScheduleTypeEnum.RECURRING;
		JobKey startJobKey = ScheduleJobHelper.generateJobKey(id, JobActionEnum.START, scheduleType);
		JobKey endJobKey = ScheduleJobHelper.generateJobKey(id, JobActionEnum.END, scheduleType);

		TriggerKey startTriggerKey = ScheduleJobHelper.generateTriggerKey(id, JobActionEnum.START, scheduleType);
		TriggerKey endTriggerKey = ScheduleJobHelper.generateTriggerKey(id, JobActionEnum.END, scheduleType);

		scheduleJobManager.createCronJob(recurringScheduleEntity);

		assertCreateCronJob(expectedCronExpressionForStartJob, expectedCronExpressionForEndJob, recurringScheduleEntity,
				startJobKey, endJobKey, startTriggerKey, endTriggerKey);
	}

	@Test
	public void testCreateCronJob_with_dayOfWeek_EuropeAmsterdam() throws Exception {
		String timeZone = "Europe/Amsterdam";

		String startTime = "22:10:00";
		String endTime = "23:20:00";
		int[] dayOfWeek = { 1, 2, 3, 4, 5, 6, 7 };

		String expectedCronExpressionForStartJob = "00 10 22 ? * MON,TUE,WED,THU,FRI,SAT,SUN *";
		String expectedCronExpressionForEndJob = "00 20 23 ? * MON,TUE,WED,THU,FRI,SAT,SUN *";

		RecurringScheduleEntity recurringScheduleEntity = createRecurringScheduleWithDaysOfWeek(timeZone, startTime,
				endTime, dayOfWeek);

		Long id = recurringScheduleEntity.getId();
		ScheduleTypeEnum scheduleType = ScheduleTypeEnum.RECURRING;
		JobKey startJobKey = ScheduleJobHelper.generateJobKey(id, JobActionEnum.START, scheduleType);
		JobKey endJobKey = ScheduleJobHelper.generateJobKey(id, JobActionEnum.END, scheduleType);

		TriggerKey startTriggerKey = ScheduleJobHelper.generateTriggerKey(id, JobActionEnum.START, scheduleType);
		TriggerKey endTriggerKey = ScheduleJobHelper.generateTriggerKey(id, JobActionEnum.END, scheduleType);

		scheduleJobManager.createCronJob(recurringScheduleEntity);

		assertCreateCronJob(expectedCronExpressionForStartJob, expectedCronExpressionForEndJob, recurringScheduleEntity,
				startJobKey, endJobKey, startTriggerKey, endTriggerKey);
	}

	@Test
	public void testCreateCronJobs_with_daysOfMonth_AmericaPhoenix() throws Exception {
		String timeZone = "America/Phoenix";

		String startTime = "00:01:00";
		String endTime = "23:59:00";
		int[] daysOfMonth = { 1, 5, 10, 20, 31 };

		String expectedCronExpressionForStartJob = "00 01 00 1,5,10,20,31 * ? *";
		String expectedCronExpressionForEndJob = "00 59 23 1,5,10,20,31 * ? *";

		RecurringScheduleEntity recurringScheduleEntity = createRecurringScheduleWithDaysOfMonth(timeZone, startTime,
				endTime, daysOfMonth);

		Long id = recurringScheduleEntity.getId();
		ScheduleTypeEnum scheduleType = ScheduleTypeEnum.RECURRING;
		JobKey startJobKey = ScheduleJobHelper.generateJobKey(id, JobActionEnum.START, scheduleType);
		JobKey endJobKey = ScheduleJobHelper.generateJobKey(id, JobActionEnum.END, scheduleType);

		TriggerKey startTriggerKey = ScheduleJobHelper.generateTriggerKey(id, JobActionEnum.START, scheduleType);
		TriggerKey endTriggerKey = ScheduleJobHelper.generateTriggerKey(id, JobActionEnum.END, scheduleType);

		scheduleJobManager.createCronJob(recurringScheduleEntity);

		assertCreateCronJob(expectedCronExpressionForStartJob, expectedCronExpressionForEndJob, recurringScheduleEntity,
				startJobKey, endJobKey, startTriggerKey, endTriggerKey);

	}

	@Test
	public void testCreateCronJobs_with_daysOfMonth_PacificKiritimati() throws Exception {
		String timeZone = "Pacific/Kiritimati";

		String startTime = "22:10:00";
		String endTime = "23:20:00";
		int[] daysOfMonth = { 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25,
				26, 27, 28, 29, 30, 31 };

		String expectedCronExpressionForStartJob = "00 10 22 1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31 * ? *";
		String expectedCronExpressionForEndJob = "00 20 23 1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31 * ? *";

		RecurringScheduleEntity recurringScheduleEntity = createRecurringScheduleWithDaysOfMonth(timeZone, startTime,
				endTime, daysOfMonth);

		Long id = recurringScheduleEntity.getId();
		ScheduleTypeEnum scheduleType = ScheduleTypeEnum.RECURRING;
		JobKey startJobKey = ScheduleJobHelper.generateJobKey(id, JobActionEnum.START, scheduleType);
		JobKey endJobKey = ScheduleJobHelper.generateJobKey(id, JobActionEnum.END, scheduleType);

		TriggerKey startTriggerKey = ScheduleJobHelper.generateTriggerKey(id, JobActionEnum.START, scheduleType);
		TriggerKey endTriggerKey = ScheduleJobHelper.generateTriggerKey(id, JobActionEnum.END, scheduleType);

		scheduleJobManager.createCronJob(recurringScheduleEntity);

		assertCreateCronJob(expectedCronExpressionForStartJob, expectedCronExpressionForEndJob, recurringScheduleEntity,
				startJobKey, endJobKey, startTriggerKey, endTriggerKey);
	}

	@Test
	public void testDeleteSimpleJobs() throws Exception {
		String appId = "appId";
		Long scheduleId = 1L;
		ScheduleTypeEnum scheduleType = ScheduleTypeEnum.SPECIFIC_DATE;

		JobKey startJobKey = ScheduleJobHelper.generateJobKey(scheduleId, JobActionEnum.START, scheduleType);
		JobKey endJobKey = ScheduleJobHelper.generateJobKey(scheduleId, JobActionEnum.END, scheduleType);

		scheduleJobManager.deleteJob(appId, scheduleId, scheduleType);

		Mockito.verify(scheduler).deleteJob(eq(startJobKey));
		Mockito.verify(scheduler).deleteJob(eq(endJobKey));
	}

	@Test
	public void testDeleteCronJobs() throws Exception {
		String appId = "appId";
		Long scheduleId = 1L;
		ScheduleTypeEnum scheduleType = ScheduleTypeEnum.RECURRING;

		JobKey startJobKey = ScheduleJobHelper.generateJobKey(scheduleId, JobActionEnum.START, scheduleType);
		JobKey endJobKey = ScheduleJobHelper.generateJobKey(scheduleId, JobActionEnum.END, scheduleType);

		scheduleJobManager.deleteJob(appId, scheduleId, scheduleType);

		Mockito.verify(scheduler).deleteJob(eq(startJobKey));
		Mockito.verify(scheduler).deleteJob(eq(endJobKey));
	}

	@Test
	public void testCreateSimpleJob_with_throw_SchedulerException_at_Quartz() throws SchedulerException {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String timeZone = TimeZone.getDefault().getID();

		SpecificDateScheduleEntity specificDateScheduleEntity = new SpecificDateScheduleEntitiesBuilder(1)
				.setAppid(appId).setTimeZone(timeZone).setScheduleId().setDefaultInstanceMinCount(1)
				.setDefaultInstanceMaxCount(5).build().get(0);

		// Set mock object for Quartz.
		Mockito.doThrow(new SchedulerException("test exception")).when(scheduler).scheduleJob(Mockito.anyObject(),
				Mockito.anyObject());

		scheduleJobManager.createSimpleJob(specificDateScheduleEntity);

		assertTrue("This test should have an Error.", validationErrorResult.hasErrors());

		List<String> errors = validationErrorResult.getAllErrorMessages();
		assertEquals(1, errors.size());

		String errorMessage = messageBundleResourceHelper.lookupMessage("scheduler.error.create.failed",
				"app_id=" + appId, "test exception");
		assertEquals(errorMessage, errors.get(0));
	}

	@Test
	public void testCreateCronJob_with_throw_SchedulerException_at_Quartz() throws SchedulerException {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String timeZone = TimeZone.getDefault().getID();

		RecurringScheduleEntity recurringScheduleEntity = new RecurringScheduleEntitiesBuilder(1, 1).setAppId(appId)
				.setTimeZone(timeZone).setScheduleId().setDefaultInstanceMinCount(1).setDefaultInstanceMaxCount(5)
				.build().get(0);

		// Set mock object for Quartz.
		Mockito.doThrow(new SchedulerException("test exception")).when(scheduler).scheduleJob(Mockito.anyObject(),
				Mockito.anyObject());

		scheduleJobManager.createCronJob(recurringScheduleEntity);

		assertTrue("This test should have an Error.", validationErrorResult.hasErrors());
		List<String> errors = validationErrorResult.getAllErrorMessages();
		assertEquals(1, errors.size());

		String errorMessage = messageBundleResourceHelper.lookupMessage("scheduler.error.create.failed",
				"app_id=" + appId, "test exception");
		assertEquals(errorMessage, errors.get(0));
	}

	@Test
	public void testDeleteSimpleJob_with_throw_SchedulerException_at_Quartz() throws SchedulerException {
		String appId = "appId";
		Long scheduleId = 1L;
		ScheduleTypeEnum type = ScheduleTypeEnum.SPECIFIC_DATE;

		Mockito.doThrow(new SchedulerException("test exception")).when(scheduler).deleteJob(Mockito.anyObject());

		scheduleJobManager.deleteJob(appId, scheduleId, type);

		assertTrue("This test should have an Error.", validationErrorResult.hasErrors());

		List<String> errors = validationErrorResult.getAllErrorMessages();
		assertEquals(1, errors.size());

		String errorMessage = messageBundleResourceHelper.lookupMessage("scheduler.error.delete.failed",
				"app_id=" + appId, "test exception");
		assertEquals(errorMessage, errors.get(0));
	}

	@Test
	public void testDeleteCronJob_with_throw_SchedulerException_at_Quartz() throws SchedulerException {
		String appId = "appId";
		Long scheduleId = 1L;
		ScheduleTypeEnum type = ScheduleTypeEnum.RECURRING;

		Mockito.doThrow(new SchedulerException("test exception")).when(scheduler).deleteJob(Mockito.anyObject());

		scheduleJobManager.deleteJob(appId, scheduleId, type);

		assertTrue("This test should have an Error.", validationErrorResult.hasErrors());
		List<String> errors = validationErrorResult.getAllErrorMessages();
		assertEquals(1, errors.size());

		String errorMessage = messageBundleResourceHelper.lookupMessage("scheduler.error.delete.failed",
				"app_id=" + appId, "test exception");
		assertEquals(errorMessage, errors.get(0));
	}

	private RecurringScheduleEntity createRecurringScheduleWithDaysOfMonth(String timeZone, String startTime,
			String endTime, int[] dayOfMonth) throws SchedulerException {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];

		Time startDateTime = Time.valueOf(startTime);
		Time endDateTime = Time.valueOf(endTime);

		return new RecurringScheduleEntitiesBuilder(1, 0).setAppId(appId).setTimeZone(timeZone).setScheduleId()
				.setDefaultInstanceMinCount(1).setDefaultInstanceMaxCount(5).setStartTime(0, startDateTime)
				.setEndTime(0, endDateTime).setDayOfMonth(0, dayOfMonth).build().get(0);
	}

	private RecurringScheduleEntity createRecurringScheduleWithDaysOfWeek(String timeZone, String startTime,
			String endTime, int[] dayOfWeek) throws SchedulerException {
		Calendar starDateCal = Calendar.getInstance();
		starDateCal.set(2080, 9, 1, 0, 0, 0);
		starDateCal.clear(Calendar.MILLISECOND);

		Calendar endDateCal = Calendar.getInstance();
		endDateCal.set(2080, 9, 20, 0, 0, 0);
		endDateCal.clear(Calendar.MILLISECOND);

		String appId = TestDataSetupHelper.generateAppIds(1)[0];

		Time startDateTime = Time.valueOf(startTime);
		Time endDateTime = Time.valueOf(endTime);

		return new RecurringScheduleEntitiesBuilder(0, 1).setAppId(appId).setTimeZone(timeZone).setScheduleId()
				.setDefaultInstanceMinCount(1).setDefaultInstanceMaxCount(5).setStartTime(0, startDateTime)
				.setEndTime(0, endDateTime).setStartDate(0, starDateCal.getTime()).setEndDate(0, endDateCal.getTime())
				.setDayOfWeek(0, dayOfWeek).build().get(0);
	}

	private void assertCreateCronJob(String expectedCronExpressionForStartJob, String expectedCronExpressionForEndJob,
			RecurringScheduleEntity recurringScheduleEntity, JobKey startJobKey, JobKey endJobKey,
			TriggerKey startTriggerKey, TriggerKey endTriggerKey) throws SchedulerException {

		assertThat("Never call it", validationErrorResult.hasErrors(), is(false));

		ArgumentCaptor<JobDetail> jobDetailArgumentCaptor = ArgumentCaptor.forClass(JobDetail.class);
		ArgumentCaptor<Trigger> triggerArgumentCaptor = ArgumentCaptor.forClass(Trigger.class);
		Mockito.verify(scheduler, Mockito.times(2)).scheduleJob(jobDetailArgumentCaptor.capture(),
				triggerArgumentCaptor.capture());

		Map<JobKey, JobDetail> scheduleJobKeyDetailMap = new HashMap<>();
		for (JobDetail jobDetail : jobDetailArgumentCaptor.getAllValues()) {
			scheduleJobKeyDetailMap.put(jobDetail.getKey(), jobDetail);
		}
		assertCreatedJobs(scheduleJobKeyDetailMap, recurringScheduleEntity, ScheduleTypeEnum.RECURRING);

		for (Trigger trigger : triggerArgumentCaptor.getAllValues()) {
			CronTrigger cronTrigger = (CronTrigger) trigger;
			if (trigger.getKey().equals(startTriggerKey)) {
				assertThat(trigger.getJobKey(), is(startJobKey));
				assertThat(cronTrigger.getCronExpression(), is(expectedCronExpressionForStartJob));
			} else if (trigger.getKey().equals(endTriggerKey)) {
				assertThat(trigger.getJobKey(), is(endJobKey));
				assertThat(cronTrigger.getCronExpression(), is(expectedCronExpressionForEndJob));
			} else {
				fail("Invalid trigger key :" + trigger.getKey());
			}

			assertThat(cronTrigger.getTimeZone(), is(TimeZone.getTimeZone(recurringScheduleEntity.getTimeZone())));
			if (recurringScheduleEntity.getStartDate() != null) {
				assertThat(cronTrigger.getStartTime(), is(recurringScheduleEntity.getStartDate()));
			}
			assertThat(cronTrigger.getEndTime(), is(recurringScheduleEntity.getEndDate()));
		}
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
		JobDataMap jobDataMap = expectedJobDetail.getJobDataMap();
		assertEquals(expectedAppId, jobDataMap.get(ScheduleJobHelper.APP_ID));
		assertEquals(expectedScheduleId, jobDataMap.get(ScheduleJobHelper.SCHEDULE_ID));
		assertEquals(expectedInstanceMinCount, jobDataMap.get(ScheduleJobHelper.INSTANCE_MIN_COUNT));
		assertEquals(expectedInstanceMaxCount, jobDataMap.get(ScheduleJobHelper.INSTANCE_MAX_COUNT));
	}

}
