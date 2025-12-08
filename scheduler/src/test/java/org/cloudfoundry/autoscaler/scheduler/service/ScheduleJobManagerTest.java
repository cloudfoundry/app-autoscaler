package org.cloudfoundry.autoscaler.scheduler.service;

import static org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper.APP_ID;
import static org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper.DEFAULT_INSTANCE_MAX_COUNT;
import static org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper.DEFAULT_INSTANCE_MIN_COUNT;
import static org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper.END_JOB_CRON_EXPRESSION;
import static org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper.END_JOB_START_TIME;
import static org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper.INSTANCE_MAX_COUNT;
import static org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper.INSTANCE_MIN_COUNT;
import static org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper.SCHEDULE_ID;
import static org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper.TIMEZONE;
import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.Matchers.is;
import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertTrue;
import static org.mockito.ArgumentMatchers.any;

import java.time.LocalDate;
import java.time.LocalDateTime;
import java.time.LocalTime;
import java.time.ZonedDateTime;
import java.util.Date;
import java.util.List;
import java.util.TimeZone;
import org.cloudfoundry.autoscaler.scheduler.dao.ActiveScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.ActiveScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.quartz.AppScalingRecurringScheduleStartJob;
import org.cloudfoundry.autoscaler.scheduler.quartz.AppScalingSpecificDateScheduleStartJob;
import org.cloudfoundry.autoscaler.scheduler.util.DateHelper;
import org.cloudfoundry.autoscaler.scheduler.util.JobActionEnum;
import org.cloudfoundry.autoscaler.scheduler.util.RecurringScheduleEntitiesBuilder;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleTypeEnum;
import org.cloudfoundry.autoscaler.scheduler.util.SpecificDateScheduleEntitiesBuilder;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataDbUtil;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.ValidationErrorResult;
import org.hamcrest.Matchers;
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
import org.quartz.SimpleTrigger;
import org.quartz.Trigger;
import org.quartz.TriggerKey;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.bean.override.mockito.MockitoBean;
import org.springframework.test.context.junit4.SpringRunner;

@RunWith(SpringRunner.class)
@SpringBootTest
public class ScheduleJobManagerTest {

  @MockitoBean private Scheduler scheduler;

  @MockitoBean private ActiveScheduleDao activeScheduleDao;

  @Autowired private ScheduleJobManager scheduleJobManager;

  @Autowired private ValidationErrorResult validationErrorResult;

  @Autowired private MessageBundleResourceHelper messageBundleResourceHelper;

  @Autowired private TestDataDbUtil testDataDbUtil;

  @Before
  public void before() throws SchedulerException {
    testDataDbUtil.cleanupData();

    Mockito.reset(scheduler);
    Mockito.reset(activeScheduleDao);
  }

  @Test
  public void testCreateSimpleJobs_with_Gmt_timeZone() throws Exception {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    TimeZone timeZone = TimeZone.getTimeZone("GMT");

    LocalDateTime startDateTime = LocalDateTime.now();
    LocalDateTime endDateTime = LocalDateTime.now().plusSeconds(10);

    SpecificDateScheduleEntity specificDateScheduleEntity =
        new SpecificDateScheduleEntitiesBuilder(1)
            .setAppid(appId)
            .setTimeZone(timeZone.getID())
            .setScheduleId()
            .setStartDateTime(0, startDateTime)
            .setEndDateTime(0, endDateTime)
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .build()
            .get(0);

    scheduleJobManager.createSimpleJob(specificDateScheduleEntity);

    Long scheduleId = specificDateScheduleEntity.getId();
    ScheduleTypeEnum scheduleType = ScheduleTypeEnum.SPECIFIC_DATE;
    String keyName = scheduleId + JobActionEnum.START.getJobIdSuffix();
    JobKey startJobKey = new JobKey(keyName, scheduleType.getScheduleIdentifier());
    TriggerKey startTriggerKey = new TriggerKey(keyName, scheduleType.getScheduleIdentifier());
    ZonedDateTime expectedStartDateTime = DateHelper.getZonedDateTime(startDateTime, timeZone);

    ArgumentCaptor<JobDetail> jobDetailArgumentCaptor = ArgumentCaptor.forClass(JobDetail.class);
    ArgumentCaptor<Trigger> triggerArgumentCaptor = ArgumentCaptor.forClass(Trigger.class);

    assertThat("No validation error", validationErrorResult.hasErrors(), is(false));

    Mockito.verify(scheduler, Mockito.times(1))
        .scheduleJob(jobDetailArgumentCaptor.capture(), triggerArgumentCaptor.capture());

    assertSimpleJobDetail(jobDetailArgumentCaptor.getValue(), specificDateScheduleEntity);

    assertSimpleTrigger(
        triggerArgumentCaptor.getValue(), expectedStartDateTime, startJobKey, startTriggerKey);
  }

  @Test
  public void testCreateCronJob_with_dayOfWeek_Gmt() throws Exception {
    String timeZone = "GMT";

    String startTime = "22:10:00";
    String endTime = "23:20:00";
    int[] dayOfWeek = {2, 4, 6};

    RecurringScheduleEntity recurringScheduleEntity =
        createRecurringScheduleWithDaysOfWeek(timeZone, startTime, endTime, dayOfWeek);

    scheduleJobManager.createCronJob(recurringScheduleEntity);

    Long scheduleId = recurringScheduleEntity.getId();
    ScheduleTypeEnum scheduleType = ScheduleTypeEnum.RECURRING;
    String keyName = scheduleId + JobActionEnum.START.getJobIdSuffix();
    JobKey startJobKey = new JobKey(keyName, scheduleType.getScheduleIdentifier());
    TriggerKey startTriggerKey = new TriggerKey(keyName, scheduleType.getScheduleIdentifier());
    String expectedCronExpressionForStartJob = "00 10 22 ? * TUE,THU,SAT *";
    String expectedCronExpressionForEndJob = "00 20 23 ? * TUE,THU,SAT *";

    ArgumentCaptor<JobDetail> jobDetailArgumentCaptor = ArgumentCaptor.forClass(JobDetail.class);
    ArgumentCaptor<Trigger> triggerArgumentCaptor = ArgumentCaptor.forClass(Trigger.class);
    Mockito.verify(scheduler, Mockito.times(1))
        .scheduleJob(jobDetailArgumentCaptor.capture(), triggerArgumentCaptor.capture());

    assertThat("No validation error", validationErrorResult.hasErrors(), is(false));

    assertCronJobDetail(
        jobDetailArgumentCaptor.getValue(),
        recurringScheduleEntity,
        expectedCronExpressionForEndJob);

    assertCronTrigger(
        triggerArgumentCaptor.getValue(),
        expectedCronExpressionForStartJob,
        recurringScheduleEntity,
        startJobKey,
        startTriggerKey);
  }

  @Test
  public void testCreateCronJob_with_dayOfWeek_EuropeAmsterdam() throws Exception {
    String timeZone = "Europe/Amsterdam";

    String startTime = "22:10:00";
    String endTime = "23:20:00";
    int[] dayOfWeek = {1, 2, 3, 4, 5, 6, 7};

    RecurringScheduleEntity recurringScheduleEntity =
        createRecurringScheduleWithDaysOfWeek(timeZone, startTime, endTime, dayOfWeek);

    scheduleJobManager.createCronJob(recurringScheduleEntity);

    Long scheduleId = recurringScheduleEntity.getId();
    ScheduleTypeEnum scheduleType = ScheduleTypeEnum.RECURRING;
    String keyName = scheduleId + JobActionEnum.START.getJobIdSuffix();
    JobKey startJobKey = new JobKey(keyName, scheduleType.getScheduleIdentifier());
    TriggerKey startTriggerKey = new TriggerKey(keyName, scheduleType.getScheduleIdentifier());
    String expectedCronExpressionForStartJob = "00 10 22 ? * MON,TUE,WED,THU,FRI,SAT,SUN *";
    String expectedCronExpressionForEndJob = "00 20 23 ? * MON,TUE,WED,THU,FRI,SAT,SUN *";

    ArgumentCaptor<JobDetail> jobDetailArgumentCaptor = ArgumentCaptor.forClass(JobDetail.class);
    ArgumentCaptor<Trigger> triggerArgumentCaptor = ArgumentCaptor.forClass(Trigger.class);
    Mockito.verify(scheduler, Mockito.times(1))
        .scheduleJob(jobDetailArgumentCaptor.capture(), triggerArgumentCaptor.capture());

    assertThat("No validation error", validationErrorResult.hasErrors(), is(false));

    assertCronJobDetail(
        jobDetailArgumentCaptor.getValue(),
        recurringScheduleEntity,
        expectedCronExpressionForEndJob);

    assertCronTrigger(
        triggerArgumentCaptor.getValue(),
        expectedCronExpressionForStartJob,
        recurringScheduleEntity,
        startJobKey,
        startTriggerKey);
  }

  @Test
  public void testCreateCronJobs_with_daysOfMonth_AmericaPhoenix() throws Exception {
    String timeZone = "America/Phoenix";

    String startTime = "00:01:00";
    String endTime = "23:59:00";
    int[] daysOfMonth = {1, 5, 10, 20, 31};

    RecurringScheduleEntity recurringScheduleEntity =
        createRecurringScheduleWithDaysOfMonth(timeZone, startTime, endTime, daysOfMonth);

    scheduleJobManager.createCronJob(recurringScheduleEntity);

    Long scheduleId = recurringScheduleEntity.getId();
    ScheduleTypeEnum scheduleType = ScheduleTypeEnum.RECURRING;
    String keyName = scheduleId + JobActionEnum.START.getJobIdSuffix();
    JobKey startJobKey = new JobKey(keyName, scheduleType.getScheduleIdentifier());
    TriggerKey startTriggerKey = new TriggerKey(keyName, scheduleType.getScheduleIdentifier());
    String expectedCronExpressionForStartJob = "00 01 00 1,5,10,20,31 * ? *";
    String expectedCronExpressionForEndJob = "00 59 23 1,5,10,20,31 * ? *";

    ArgumentCaptor<JobDetail> jobDetailArgumentCaptor = ArgumentCaptor.forClass(JobDetail.class);
    ArgumentCaptor<Trigger> triggerArgumentCaptor = ArgumentCaptor.forClass(Trigger.class);
    Mockito.verify(scheduler, Mockito.times(1))
        .scheduleJob(jobDetailArgumentCaptor.capture(), triggerArgumentCaptor.capture());

    assertThat("No validation error", validationErrorResult.hasErrors(), is(false));

    assertCronJobDetail(
        jobDetailArgumentCaptor.getValue(),
        recurringScheduleEntity,
        expectedCronExpressionForEndJob);

    assertCronTrigger(
        triggerArgumentCaptor.getValue(),
        expectedCronExpressionForStartJob,
        recurringScheduleEntity,
        startJobKey,
        startTriggerKey);
  }

  @Test
  public void testCreateCronJobs_with_daysOfMonth_PacificKiritimati() throws Exception {
    String timeZone = "Pacific/Kiritimati";

    String startTime = "22:10:00";
    String endTime = "23:20:00";
    int[] daysOfMonth = {
      1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
      27, 28, 29, 30, 31
    };

    RecurringScheduleEntity recurringScheduleEntity =
        createRecurringScheduleWithDaysOfMonth(timeZone, startTime, endTime, daysOfMonth);

    scheduleJobManager.createCronJob(recurringScheduleEntity);

    Long scheduleId = recurringScheduleEntity.getId();
    ScheduleTypeEnum scheduleType = ScheduleTypeEnum.RECURRING;
    String keyName = scheduleId + JobActionEnum.START.getJobIdSuffix();
    JobKey startJobKey = new JobKey(keyName, scheduleType.getScheduleIdentifier());
    TriggerKey startTriggerKey = new TriggerKey(keyName, scheduleType.getScheduleIdentifier());
    String expectedCronExpressionForStartJob =
        "00 10 22"
            + " 1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31"
            + " * ? *";
    String expectedCronExpressionForEndJob =
        "00 20 23"
            + " 1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31"
            + " * ? *";

    ArgumentCaptor<JobDetail> jobDetailArgumentCaptor = ArgumentCaptor.forClass(JobDetail.class);
    ArgumentCaptor<Trigger> triggerArgumentCaptor = ArgumentCaptor.forClass(Trigger.class);
    Mockito.verify(scheduler, Mockito.times(1))
        .scheduleJob(jobDetailArgumentCaptor.capture(), triggerArgumentCaptor.capture());

    assertThat("No validation error", validationErrorResult.hasErrors(), is(false));

    assertCronJobDetail(
        jobDetailArgumentCaptor.getValue(),
        recurringScheduleEntity,
        expectedCronExpressionForEndJob);

    assertCronTrigger(
        triggerArgumentCaptor.getValue(),
        expectedCronExpressionForStartJob,
        recurringScheduleEntity,
        startJobKey,
        startTriggerKey);
  }

  @Test
  public void testDeleteSimpleJobs() throws Exception {
    String appId = "appId";
    Long scheduleId = 1L;
    ScheduleTypeEnum scheduleType = ScheduleTypeEnum.SPECIFIC_DATE;
    Long startJobIdentifier = System.currentTimeMillis();

    ActiveScheduleEntity activeScheduleEntity = new ActiveScheduleEntity();
    activeScheduleEntity.setStartJobIdentifier(scheduleId);
    activeScheduleEntity.setAppId(appId);
    activeScheduleEntity.setStartJobIdentifier(startJobIdentifier);
    Mockito.when(activeScheduleDao.find(scheduleId)).thenReturn(activeScheduleEntity);

    scheduleJobManager.deleteJob(appId, scheduleId, scheduleType);

    JobKey startJobKey =
        new JobKey(
            scheduleId + JobActionEnum.START.getJobIdSuffix(),
            scheduleType.getScheduleIdentifier());
    JobKey endJobKey =
        new JobKey(
            scheduleId + JobActionEnum.END.getJobIdSuffix() + "_" + startJobIdentifier, "Schedule");

    Mockito.verify(scheduler, Mockito.times(1)).deleteJob(startJobKey);
    Mockito.verify(scheduler, Mockito.times(1)).deleteJob(endJobKey);
  }

  @Test
  public void testDeleteCronJobs() throws Exception {
    String appId = "appId";
    Long scheduleId = 1L;
    ScheduleTypeEnum scheduleType = ScheduleTypeEnum.RECURRING;
    Long startJobIdentifier = System.currentTimeMillis();

    ActiveScheduleEntity activeScheduleEntity = new ActiveScheduleEntity();
    activeScheduleEntity.setStartJobIdentifier(scheduleId);
    activeScheduleEntity.setAppId(appId);
    activeScheduleEntity.setStartJobIdentifier(startJobIdentifier);
    Mockito.when(activeScheduleDao.find(scheduleId)).thenReturn(activeScheduleEntity);

    scheduleJobManager.deleteJob(appId, scheduleId, scheduleType);

    JobKey startJobKey =
        new JobKey(
            scheduleId + JobActionEnum.START.getJobIdSuffix(),
            scheduleType.getScheduleIdentifier());
    JobKey endJobKey =
        new JobKey(
            scheduleId + JobActionEnum.END.getJobIdSuffix() + "_" + startJobIdentifier, "Schedule");

    Mockito.verify(scheduler, Mockito.times(1)).deleteJob(startJobKey);
    Mockito.verify(scheduler, Mockito.times(1)).deleteJob(endJobKey);
  }

  @Test
  public void testDeleteSimpleJobs_without_activeSchedule() throws Exception {
    String appId = "appId";
    Long scheduleId = 1L;
    ScheduleTypeEnum scheduleType = ScheduleTypeEnum.SPECIFIC_DATE;

    scheduleJobManager.deleteJob(appId, scheduleId, scheduleType);

    Mockito.verify(scheduler, Mockito.times(1)).deleteJob(any());
  }

  @Test
  public void testCreateSimpleJob_with_throw_SchedulerException_at_Quartz()
      throws SchedulerException {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String timeZone = TimeZone.getDefault().getID();

    SpecificDateScheduleEntity specificDateScheduleEntity =
        new SpecificDateScheduleEntitiesBuilder(1)
            .setAppid(appId)
            .setTimeZone(timeZone)
            .setScheduleId()
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .build()
            .get(0);

    // Set mock object for Quartz.
    Mockito.doThrow(new SchedulerException("test exception"))
        .when(scheduler)
        .scheduleJob(any(), any());

    scheduleJobManager.createSimpleJob(specificDateScheduleEntity);

    assertTrue("This test should have an Error.", validationErrorResult.hasErrors());

    List<String> errors = validationErrorResult.getAllErrorMessages();
    assertEquals(1, errors.size());

    String errorMessage =
        messageBundleResourceHelper.lookupMessage(
            "scheduler.error.create.failed", "app_id=" + appId, "test exception");
    assertEquals(errorMessage, errors.get(0));
  }

  @Test
  public void testCreateCronJob_with_throw_SchedulerException_at_Quartz()
      throws SchedulerException {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String timeZone = TimeZone.getDefault().getID();

    RecurringScheduleEntity recurringScheduleEntity =
        new RecurringScheduleEntitiesBuilder(1, 1)
            .setAppId(appId)
            .setTimeZone(timeZone)
            .setScheduleId()
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .build()
            .get(0);

    // Set mock object for Quartz.
    Mockito.doThrow(new SchedulerException("test exception"))
        .when(scheduler)
        .scheduleJob(any(), any());

    scheduleJobManager.createCronJob(recurringScheduleEntity);

    List<String> errors = validationErrorResult.getAllErrorMessages();

    String errorMessage =
        messageBundleResourceHelper.lookupMessage(
            "scheduler.error.create.failed", "app_id=" + appId, "test exception");
    assertEquals(errorMessage, errors.get(0));
    assertEquals(1, errors.size());
    assertTrue("This test should have an Error.", validationErrorResult.hasErrors());
  }

  @Test
  public void testDeleteSimpleJob_with_throw_SchedulerException_at_Quartz()
      throws SchedulerException {
    String appId = "appId";
    Long scheduleId = 1L;
    ScheduleTypeEnum type = ScheduleTypeEnum.SPECIFIC_DATE;

    Mockito.doThrow(new SchedulerException("test exception")).when(scheduler).deleteJob(any());

    scheduleJobManager.deleteJob(appId, scheduleId, type);

    List<String> errors = validationErrorResult.getAllErrorMessages();

    String errorMessage =
        messageBundleResourceHelper.lookupMessage(
            "scheduler.error.delete.failed", "app_id=" + appId, "test exception");
    assertEquals(errorMessage, errors.get(0));
    assertEquals(1, errors.size());
    assertTrue("This test should have an Error.", validationErrorResult.hasErrors());
  }

  @Test
  public void testDeleteCronJob_with_throw_SchedulerException_at_Quartz()
      throws SchedulerException {
    String appId = "appId";
    Long scheduleId = 1L;
    ScheduleTypeEnum type = ScheduleTypeEnum.RECURRING;

    Mockito.doThrow(new SchedulerException("test exception")).when(scheduler).deleteJob(any());

    scheduleJobManager.deleteJob(appId, scheduleId, type);

    List<String> errors = validationErrorResult.getAllErrorMessages();

    String errorMessage =
        messageBundleResourceHelper.lookupMessage(
            "scheduler.error.delete.failed", "app_id=" + appId, "test exception");
    assertEquals(errorMessage, errors.get(0));
    assertEquals(1, errors.size());
    assertTrue("This test should have an Error.", validationErrorResult.hasErrors());
  }

  private RecurringScheduleEntity createRecurringScheduleWithDaysOfMonth(
      String timeZone, String startTime, String endTime, int[] dayOfMonth)
      throws SchedulerException {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];

    LocalTime startDateTime = LocalTime.parse(startTime);
    LocalTime endDateTime = LocalTime.parse(endTime);

    return new RecurringScheduleEntitiesBuilder(1, 0)
        .setAppId(appId)
        .setTimeZone(timeZone)
        .setScheduleId()
        .setDefaultInstanceMinCount(1)
        .setDefaultInstanceMaxCount(5)
        .setStartTime(0, startDateTime)
        .setEndTime(0, endDateTime)
        .setDayOfMonth(0, dayOfMonth)
        .build()
        .get(0);
  }

  private RecurringScheduleEntity createRecurringScheduleWithDaysOfWeek(
      String timeZone, String startTime, String endTime, int[] dayOfWeek)
      throws SchedulerException {

    LocalDate startDate = LocalDate.of(2080, 9, 1);
    LocalDate endDate = LocalDate.of(2080, 9, 10);

    String appId = TestDataSetupHelper.generateAppIds(1)[0];

    LocalTime startDateTime = LocalTime.parse(startTime);
    LocalTime endDateTime = LocalTime.parse(endTime);

    return new RecurringScheduleEntitiesBuilder(0, 1)
        .setAppId(appId)
        .setTimeZone(timeZone)
        .setScheduleId()
        .setDefaultInstanceMinCount(1)
        .setDefaultInstanceMaxCount(5)
        .setStartTime(0, startDateTime)
        .setEndTime(0, endDateTime)
        .setStartDate(0, startDate)
        .setEndDate(0, endDate)
        .setDayOfWeek(0, dayOfWeek)
        .build()
        .get(0);
  }

  private void assertSimpleTrigger(
      Trigger trigger,
      ZonedDateTime expectedStartDateTime,
      JobKey expectedStartJobKey,
      TriggerKey expectedStartTriggerKey) {

    assertThat(trigger.getJobKey(), is(expectedStartJobKey));
    assertThat(trigger.getKey(), is(expectedStartTriggerKey));
    assertThat(trigger.getStartTime(), is(Date.from(expectedStartDateTime.toInstant())));
    assertThat(trigger.getMisfireInstruction(), is(SimpleTrigger.MISFIRE_INSTRUCTION_FIRE_NOW));
  }

  private void assertCronTrigger(
      Trigger trigger,
      String expectedCronExpressionForStartJob,
      RecurringScheduleEntity recurringScheduleEntity,
      JobKey startJobKey,
      TriggerKey startTriggerKey) {

    CronTrigger cronTrigger = (CronTrigger) trigger;
    assertThat(cronTrigger.getJobKey(), is(startJobKey));
    assertThat(cronTrigger.getKey(), is(startTriggerKey));
    assertThat(cronTrigger.getCronExpression(), is(expectedCronExpressionForStartJob));
    assertThat(
        cronTrigger.getTimeZone(), is(TimeZone.getTimeZone(recurringScheduleEntity.getTimeZone())));
    assertThat(
        cronTrigger.getMisfireInstruction(), is(CronTrigger.MISFIRE_INSTRUCTION_FIRE_ONCE_NOW));

    TimeZone timeZone = TimeZone.getTimeZone(recurringScheduleEntity.getTimeZone());

    if (recurringScheduleEntity.getStartDate() != null) {
      assertThat(
          cronTrigger.getStartTime(),
          is(
              Date.from(
                  recurringScheduleEntity
                      .getStartDate()
                      .atStartOfDay(timeZone.toZoneId())
                      .toInstant())));
    }
    if (recurringScheduleEntity.getEndDate() != null) {
      assertThat(
          cronTrigger.getEndTime(),
          is(
              Date.from(
                  recurringScheduleEntity
                      .getEndDate()
                      .plusDays(1)
                      .atStartOfDay(timeZone.toZoneId())
                      .toInstant())));
    }
  }

  private void assertCommonJobDataMap(JobDataMap jobDataMap, ScheduleEntity scheduleEntity) {
    assertThat(jobDataMap.getString(APP_ID), is(scheduleEntity.getAppId()));
    assertThat(jobDataMap.getLong(SCHEDULE_ID), is(scheduleEntity.getId()));
    assertThat(jobDataMap.getString(TIMEZONE), is(scheduleEntity.getTimeZone()));
    assertThat(jobDataMap.getInt(INSTANCE_MIN_COUNT), is(scheduleEntity.getInstanceMinCount()));
    assertThat(jobDataMap.getInt(INSTANCE_MAX_COUNT), is(scheduleEntity.getInstanceMaxCount()));
    assertThat(
        jobDataMap.getInt(DEFAULT_INSTANCE_MIN_COUNT),
        is(scheduleEntity.getDefaultInstanceMinCount()));
    assertThat(
        jobDataMap.getInt(DEFAULT_INSTANCE_MAX_COUNT),
        is(scheduleEntity.getDefaultInstanceMaxCount()));
  }

  private void assertSimpleJobDetail(JobDetail jobDetail, SpecificDateScheduleEntity scheduleEntity)
      throws SchedulerException {
    assertThat("Expected existing jobDetail", jobDetail, Matchers.notNullValue());
    assertEquals(AppScalingSpecificDateScheduleStartJob.class, jobDetail.getJobClass());

    JobDataMap jobDataMap = jobDetail.getJobDataMap();
    assertCommonJobDataMap(jobDataMap, scheduleEntity);
    assertThat(jobDataMap.get(END_JOB_START_TIME), is(scheduleEntity.getEndDateTime()));
  }

  private void assertCronJobDetail(
      JobDetail jobDetail, ScheduleEntity scheduleEntity, String cronExpression)
      throws SchedulerException {
    assertThat("Expected existing jobDetail", jobDetail, Matchers.notNullValue());
    assertEquals(AppScalingRecurringScheduleStartJob.class, jobDetail.getJobClass());

    JobDataMap jobDataMap = jobDetail.getJobDataMap();
    assertCommonJobDataMap(jobDataMap, scheduleEntity);
    assertThat(jobDataMap.getString(END_JOB_CRON_EXPRESSION), is(cronExpression));
  }
}
