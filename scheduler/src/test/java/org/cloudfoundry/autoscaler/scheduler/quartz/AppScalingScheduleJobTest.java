package org.cloudfoundry.autoscaler.scheduler.quartz;

import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.core.Is.is;
import static org.junit.Assert.assertTrue;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.ArgumentMatchers.notNull;

import java.io.IOException;
import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.Date;
import java.util.List;
import java.util.TimeZone;
import java.util.concurrent.TimeUnit;
import org.cloudfoundry.autoscaler.scheduler.dao.ActiveScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.ActiveScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.EmbeddedTomcatUtil;
import org.cloudfoundry.autoscaler.scheduler.util.JobActionEnum;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataDbUtil;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper.JobInformation;
import org.cloudfoundry.autoscaler.scheduler.util.TestJobListener;
import org.cloudfoundry.autoscaler.scheduler.util.error.DatabaseValidationException;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
import org.junit.After;
import org.junit.AfterClass;
import org.junit.Before;
import org.junit.BeforeClass;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.ArgumentCaptor;
import org.mockito.Mockito;
import org.mockito.MockitoAnnotations;
import org.quartz.CronExpression;
import org.quartz.JobDataMap;
import org.quartz.JobDetail;
import org.quartz.JobKey;
import org.quartz.Scheduler;
import org.quartz.SchedulerException;
import org.quartz.SimpleTrigger;
import org.quartz.Trigger;
import org.quartz.TriggerKey;
import org.quartz.impl.StdSchedulerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.context.ApplicationContext;
import org.springframework.http.HttpEntity;
import org.springframework.test.context.bean.override.mockito.MockitoBean;
import org.springframework.test.context.bean.override.mockito.MockitoSpyBean;
import org.springframework.test.context.junit4.SpringRunner;
import org.springframework.web.client.ResourceAccessException;
import org.springframework.web.client.RestOperations;

@RunWith(SpringRunner.class)
@SpringBootTest
public class AppScalingScheduleJobTest {

  private static EmbeddedTomcatUtil embeddedTomcatUtil;
  @Autowired private MessageBundleResourceHelper messageBundleResourceHelper;
  private Scheduler memScheduler;
  @MockitoBean private Scheduler scheduler;
  @MockitoBean private ActiveScheduleDao activeScheduleDao;
  @MockitoSpyBean private RestOperations restOperations;
  @Autowired private TestDataDbUtil testDataDbUtil;
  @Autowired private ApplicationContext applicationContext;

  @Value("${autoscaler.scalingengine.url}")
  private String scalingEngineUrl;

  private AutoCloseable mock;

  @BeforeClass
  public static void beforeClass() throws IOException {
    embeddedTomcatUtil = new EmbeddedTomcatUtil();
    embeddedTomcatUtil.start();
  }

  @AfterClass
  public static void afterClass() throws IOException, InterruptedException {
    embeddedTomcatUtil.stop();
  }

  @After
  public void closeMock() throws Exception {
    mock.close();
  }

  @Before
  public void before() throws SchedulerException {
    mock = MockitoAnnotations.openMocks(this);
    memScheduler = createMemScheduler();
    testDataDbUtil.cleanupData(memScheduler);

    Mockito.reset(activeScheduleDao);
    Mockito.reset(restOperations);
    Mockito.reset(scheduler);
  }

  private Scheduler createMemScheduler() throws SchedulerException {
    Scheduler scheduler = StdSchedulerFactory.getDefaultScheduler();

    QuartzJobFactory jobFactory = new QuartzJobFactory();
    jobFactory.setApplicationContext(applicationContext);
    scheduler.setJobFactory(jobFactory);

    scheduler.start();
    return scheduler;
  }

  @Test
  public void testNotifyStartOfActiveScheduleToScalingEngine_with_SpecificDateSchedule()
      throws Exception {
    // Build the job and trigger
    JobInformation jobInformation =
        new JobInformation<>(AppScalingSpecificDateScheduleStartJob.class);
    Date endJobStartTime = TestDataSetupHelper.getCurrentDateTime(1);
    JobDataMap jobDataMap =
        setupJobDataForSpecificDateSchedule(
            jobInformation.getJobDetail(), endJobStartTime, TimeZone.getDefault());

    ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
    String appId = activeScheduleEntity.getAppId();
    Long scheduleId = activeScheduleEntity.getId();

    embeddedTomcatUtil.addScalingEngineMockForAppAndScheduleId(appId, scheduleId, 200, null);

    TestJobListener testJobListener = new TestJobListener(1);
    memScheduler.getListenerManager().addJobListener(testJobListener);

    memScheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

    testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

    Mockito.verify(activeScheduleDao, Mockito.times(1)).deleteActiveSchedulesByAppId(appId);
    Mockito.verify(activeScheduleDao, Mockito.times(1)).create(any());

    // For end job
    ArgumentCaptor<JobDetail> jobDetailArgumentCaptor = ArgumentCaptor.forClass(JobDetail.class);
    ArgumentCaptor<Trigger> triggerArgumentCaptor = ArgumentCaptor.forClass(Trigger.class);

    Mockito.verify(scheduler, Mockito.times(1))
        .scheduleJob(jobDetailArgumentCaptor.capture(), triggerArgumentCaptor.capture());

    JobDataMap actualJobDataMap = jobDetailArgumentCaptor.getValue().getJobDataMap();
    assertTrue(
        actualJobDataMap.getBoolean(ScheduleJobHelper.ACTIVE_SCHEDULE_TABLE_CREATE_TASK_DONE));
    assertTrue(actualJobDataMap.getBoolean(ScheduleJobHelper.CREATE_END_JOB_TASK_DONE));

    Long startJobIdentifier =
        jobDetailArgumentCaptor
            .getValue()
            .getJobDataMap()
            .getLong(ScheduleJobHelper.START_JOB_IDENTIFIER);

    assertEndJobArgument(
        triggerArgumentCaptor.getValue(), endJobStartTime, scheduleId, startJobIdentifier);

    // For notify to Scaling Engine
    assertNotifyScalingEngineForStartJob(activeScheduleEntity, startJobIdentifier);
  }

  @Test
  public void
      testNotifyStartOfActiveScheduleToScalingEngine_with_SpecificDateSchedule_starting_after_endTime()
          throws Exception {
    // Build the job and trigger
    JobInformation jobInformation =
        new JobInformation<>(AppScalingSpecificDateScheduleStartJob.class);
    Date endJobStartTime = TestDataSetupHelper.getCurrentDateTime(-1);
    JobDataMap jobDataMap =
        setupJobDataForSpecificDateSchedule(
            jobInformation.getJobDetail(), endJobStartTime, TimeZone.getDefault());

    ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
    String appId = activeScheduleEntity.getAppId();
    Long scheduleId = activeScheduleEntity.getId();

    embeddedTomcatUtil.addScalingEngineMockForAppAndScheduleId(appId, scheduleId, 200, null);

    TestJobListener testJobListener = new TestJobListener(1);
    memScheduler.getListenerManager().addJobListener(testJobListener);

    memScheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

    testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

    Mockito.verify(activeScheduleDao, Mockito.never())
        .deleteActiveSchedulesByAppId(Mockito.anyString());
    Mockito.verify(activeScheduleDao, Mockito.never()).create(any());

    // For end job
    Mockito.verify(scheduler, Mockito.never()).scheduleJob(any(), any());

    // For notify to Scaling Engine
    Mockito.verify(restOperations, Mockito.never()).put(Mockito.anyString(), notNull());
  }

  @Test
  public void testNotifyStartOfActiveScheduleToScalingEngine_with_RecurringSchedule()
      throws Exception {
    // Build the job and trigger
    JobInformation jobInformation = new JobInformation<>(AppScalingRecurringScheduleStartJob.class);
    CronExpression endJobCronExpression = new CronExpression("00 00 00 1 * ? 2099");
    JobDataMap jobDataMap =
        setupJobDataForRecurringSchedule(
            jobInformation.getJobDetail(), endJobCronExpression.getCronExpression());

    ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
    String appId = activeScheduleEntity.getAppId();
    Long scheduleId = activeScheduleEntity.getId();

    embeddedTomcatUtil.addScalingEngineMockForAppAndScheduleId(appId, scheduleId, 200, null);

    TestJobListener testJobListener = new TestJobListener(1);
    memScheduler.getListenerManager().addJobListener(testJobListener);

    memScheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

    testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

    Mockito.verify(activeScheduleDao, Mockito.times(1)).deleteActiveSchedulesByAppId(appId);
    Mockito.verify(activeScheduleDao, Mockito.times(1)).create(any());

    // For end job
    ArgumentCaptor<JobDetail> jobDetailArgumentCaptor = ArgumentCaptor.forClass(JobDetail.class);
    ArgumentCaptor<Trigger> triggerArgumentCaptor = ArgumentCaptor.forClass(Trigger.class);

    Mockito.verify(scheduler, Mockito.times(1))
        .scheduleJob(jobDetailArgumentCaptor.capture(), triggerArgumentCaptor.capture());

    Long startJobIdentifier =
        jobDetailArgumentCaptor
            .getValue()
            .getJobDataMap()
            .getLong(ScheduleJobHelper.START_JOB_IDENTIFIER);

    assertEndJobArgument(
        triggerArgumentCaptor.getValue(),
        endJobCronExpression.getNextValidTimeAfter(new Date()),
        scheduleId,
        startJobIdentifier);

    // For notify to Scaling Engine
    assertNotifyScalingEngineForStartJob(activeScheduleEntity, startJobIdentifier);
  }

  @Test
  public void
      testNotifyStartOfActiveScheduleToScalingEngine_with_RecurringSchedule_throw_ParseException()
          throws Exception {

    // Build the job and trigger
    JobInformation jobInformation = new JobInformation<>(AppScalingRecurringScheduleStartJob.class);
    JobDataMap jobDataMap = setupJobDataForRecurringSchedule(jobInformation.getJobDetail(), null);

    jobDataMap.put(ScheduleJobHelper.END_JOB_CRON_EXPRESSION, "Invalid cron expression");

    ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
    String appId = activeScheduleEntity.getAppId();
    Long scheduleId = activeScheduleEntity.getId();

    embeddedTomcatUtil.addScalingEngineMockForAppAndScheduleId(appId, scheduleId, 200, null);

    TestJobListener testJobListener = new TestJobListener(1);
    memScheduler.getListenerManager().addJobListener(testJobListener);

    memScheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

    testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

    Mockito.verify(activeScheduleDao, Mockito.never())
        .deleteActiveSchedulesByAppId(Mockito.anyString());
    Mockito.verify(activeScheduleDao, Mockito.never()).create(any());

    // For end job
    Mockito.verify(scheduler, Mockito.never()).scheduleJob(any(), any());

    // For notify to Scaling Engine
    Mockito.verify(restOperations, Mockito.never()).put(Mockito.anyString(), Mockito.notNull());
  }

  @Test
  public void testCreateActiveSpecificDateScheduleFailed_with_existing_ActiveSchedule()
      throws Exception {
    // Build the job and trigger
    JobInformation jobInformation =
        new JobInformation<>(AppScalingSpecificDateScheduleStartJob.class);
    Date endJobStartTime = TestDataSetupHelper.getCurrentDateTime(1);
    JobDataMap jobDataMap =
        setupJobDataForSpecificDateSchedule(
            jobInformation.getJobDetail(), endJobStartTime, TimeZone.getDefault());
    ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
    String appId = activeScheduleEntity.getAppId();
    Long scheduleId = activeScheduleEntity.getId();

    embeddedTomcatUtil.addScalingEngineMockForAppAndScheduleId(appId, scheduleId, 200, null);
    TestJobListener testJobListener = new TestJobListener(1);
    memScheduler.getListenerManager().addJobListener(testJobListener);

    List<ActiveScheduleEntity> existingActiveSchedule = new ArrayList<>();
    ActiveScheduleEntity existingActiveScheduleEntity =
        ScheduleJobHelper.setupActiveSchedule(jobDataMap);
    existingActiveSchedule.add(existingActiveScheduleEntity);

    Mockito.when(activeScheduleDao.findByAppId(Mockito.anyString()))
        .thenReturn(existingActiveSchedule);
    Mockito.when(activeScheduleDao.deleteActiveSchedulesByAppId(Mockito.anyString())).thenReturn(1);

    memScheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

    testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

    Mockito.verify(activeScheduleDao, Mockito.times(1)).findByAppId(appId);
    Mockito.verify(activeScheduleDao, Mockito.times(0)).deleteActiveSchedulesByAppId(appId);
    Mockito.verify(activeScheduleDao, Mockito.times(0)).create(any());

    // For end job
    Mockito.verify(scheduler, Mockito.never()).scheduleJob(any(), any());
    Mockito.verify(restOperations, Mockito.never()).put(Mockito.anyString(), notNull());
  }

  @Test
  public void testCreateActiveScheduleFailed_with_existing_ActiveSchedule() throws Exception {
    // Build the job and trigger
    JobInformation jobInformation = new JobInformation<>(AppScalingRecurringScheduleStartJob.class);
    CronExpression endJobCronExpression = new CronExpression("00 00 00 1 * ? 2099");
    JobDataMap jobDataMap =
        setupJobDataForRecurringSchedule(
            jobInformation.getJobDetail(), endJobCronExpression.getCronExpression());
    ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
    String appId = activeScheduleEntity.getAppId();
    Long scheduleId = activeScheduleEntity.getId();

    List<ActiveScheduleEntity> existingActiveSchedule = new ArrayList<>();
    ActiveScheduleEntity existingActiveScheduleEntity =
        ScheduleJobHelper.setupActiveSchedule(jobDataMap);
    existingActiveSchedule.add(existingActiveScheduleEntity);

    Mockito.when(activeScheduleDao.findByAppId(Mockito.anyString()))
        .thenReturn(existingActiveSchedule);
    Mockito.when(activeScheduleDao.deleteActiveSchedulesByAppId(Mockito.anyString())).thenReturn(1);

    embeddedTomcatUtil.addScalingEngineMockForAppAndScheduleId(appId, scheduleId, 200, null);
    TestJobListener testJobListener = new TestJobListener(1);
    memScheduler.getListenerManager().addJobListener(testJobListener);

    memScheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

    testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

    Mockito.verify(activeScheduleDao, Mockito.times(1)).findByAppId(appId);
    Mockito.verify(activeScheduleDao, Mockito.times(0)).deleteActiveSchedulesByAppId(appId);
    Mockito.verify(activeScheduleDao, Mockito.times(0)).create(any());

    // For end job
    Mockito.verify(scheduler, Mockito.never()).scheduleJob(any(), any());
    Mockito.verify(restOperations, Mockito.never()).put(Mockito.anyString(), notNull());
  }

  @Test
  public void
      testCreateActiveScheduleFailed_with_existing_ActiveSchedule_DatabaseValidationException()
          throws Exception {

    // Build the job and trigger
    JobInformation jobInformation =
        new JobInformation<>(AppScalingSpecificDateScheduleStartJob.class);
    Date endJobStartTime = TestDataSetupHelper.getCurrentDateTime(1);

    JobDataMap jobDataMap =
        setupJobDataForSpecificDateSchedule(
            jobInformation.getJobDetail(), endJobStartTime, TimeZone.getDefault());

    ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
    String appId = activeScheduleEntity.getAppId();
    Long scheduleId = activeScheduleEntity.getId();

    embeddedTomcatUtil.addScalingEngineMockForAppAndScheduleId(appId, scheduleId, 200, null);
    Mockito.when(activeScheduleDao.findByAppId(Mockito.anyString()))
        .thenThrow(new DatabaseValidationException("test exception"));

    TestJobListener testJobListener = new TestJobListener(1);
    memScheduler.getListenerManager().addJobListener(testJobListener);

    memScheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());
    testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

    Mockito.verify(activeScheduleDao, Mockito.times(1)).findByAppId(appId);
    Mockito.verify(activeScheduleDao, Mockito.times(0)).deleteActiveSchedulesByAppId(any());
    Mockito.verify(activeScheduleDao, Mockito.never()).create(any());

    // For end job
    Mockito.verify(scheduler, Mockito.never()).scheduleJob(any(), any());
    Mockito.verify(restOperations, Mockito.never()).put(Mockito.anyString(), notNull());
  }

  @Test
  public void testNotifyEndOfActiveScheduleToScalingEngine() throws Exception {
    // Build the job
    JobInformation jobInformation = new JobInformation<>(AppScalingScheduleEndJob.class);
    Date endJobStartTime = TestDataSetupHelper.getCurrentDateTime(1);

    long startJobIdentifier = 10L;
    JobDataMap jobDataMap =
        setupJobDataForSpecificDateSchedule(
            jobInformation.getJobDetail(), endJobStartTime, TimeZone.getDefault());
    jobDataMap.put(ScheduleJobHelper.START_JOB_IDENTIFIER, startJobIdentifier);

    ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
    String appId = activeScheduleEntity.getAppId();
    Long scheduleId = activeScheduleEntity.getId();

    embeddedTomcatUtil.addScalingEngineMockForAppAndScheduleId(appId, scheduleId, 204, null);

    TestJobListener testJobListener = new TestJobListener(1);
    memScheduler.getListenerManager().addJobListener(testJobListener);

    memScheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

    testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

    Mockito.verify(activeScheduleDao, Mockito.times(1))
        .delete(activeScheduleEntity.getId(), startJobIdentifier);

    // For notify to Scaling Engine
    assertNotifyScalingEngineForEndJob(activeScheduleEntity);
  }

  @Test
  public void testCreateActiveSchedules_throw_DatabaseValidationException() throws Exception {
    int expectedNumOfTimesJobRescheduled = 2;

    // Build the job
    JobInformation jobInformation =
        new JobInformation<>(AppScalingSpecificDateScheduleStartJob.class);
    Date endJobStartTime = TestDataSetupHelper.getCurrentDateTime(1);
    JobDataMap jobDataMap =
        setupJobDataForSpecificDateSchedule(
            jobInformation.getJobDetail(), endJobStartTime, TimeZone.getDefault());

    ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
    String appId = activeScheduleEntity.getAppId();
    Long scheduleId = activeScheduleEntity.getId();

    embeddedTomcatUtil.addScalingEngineMockForAppAndScheduleId(appId, scheduleId, 200, null);

    Mockito.doThrow(new DatabaseValidationException("test exception"))
        .doNothing()
        .when(activeScheduleDao)
        .create(any());

    TestJobListener testJobListener = new TestJobListener(expectedNumOfTimesJobRescheduled);
    memScheduler.getListenerManager().addJobListener(testJobListener);

    memScheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

    testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

    Mockito.verify(activeScheduleDao, Mockito.times(expectedNumOfTimesJobRescheduled))
        .deleteActiveSchedulesByAppId(appId);
    Mockito.verify(activeScheduleDao, Mockito.times(expectedNumOfTimesJobRescheduled))
        .create(any());

    // For end job
    ArgumentCaptor<JobDetail> jobDetailArgumentCaptor = ArgumentCaptor.forClass(JobDetail.class);
    ArgumentCaptor<Trigger> triggerArgumentCaptor = ArgumentCaptor.forClass(Trigger.class);

    Mockito.verify(scheduler, Mockito.times(1))
        .scheduleJob(jobDetailArgumentCaptor.capture(), triggerArgumentCaptor.capture());

    Long startJobIdentifier =
        jobDetailArgumentCaptor
            .getValue()
            .getJobDataMap()
            .getLong(ScheduleJobHelper.START_JOB_IDENTIFIER);

    assertEndJobArgument(
        triggerArgumentCaptor.getValue(), endJobStartTime, scheduleId, startJobIdentifier);

    // For notify to Scaling Engine
    assertNotifyScalingEngineForStartJob(activeScheduleEntity, startJobIdentifier);
  }

  @Test
  public void testRemoveActiveSchedules_throw_DatabaseValidationException() throws Exception {
    int expectedNumOfTimesJobRescheduled = 2;

    // Build the job
    JobInformation jobInformation = new JobInformation<>(AppScalingScheduleEndJob.class);
    Date endJobStartTime = TestDataSetupHelper.getCurrentDateTime(1);

    long startJobIdentifier = 10L;
    JobDataMap jobDataMap =
        setupJobDataForSpecificDateSchedule(
            jobInformation.getJobDetail(), endJobStartTime, TimeZone.getDefault());
    jobDataMap.put(ScheduleJobHelper.START_JOB_IDENTIFIER, startJobIdentifier);

    ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
    String appId = activeScheduleEntity.getAppId();
    Long scheduleId = activeScheduleEntity.getId();

    embeddedTomcatUtil.addScalingEngineMockForAppAndScheduleId(appId, scheduleId, 204, null);

    Mockito.doThrow(new DatabaseValidationException("test exception"))
        .doReturn(1)
        .when(activeScheduleDao)
        .delete(eq(scheduleId), any());

    TestJobListener testJobListener = new TestJobListener(expectedNumOfTimesJobRescheduled);
    memScheduler.getListenerManager().addJobListener(testJobListener);

    memScheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

    testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

    Mockito.verify(activeScheduleDao, Mockito.times(expectedNumOfTimesJobRescheduled))
        .delete(scheduleId, startJobIdentifier);

    // For notify to Scaling Engine
    assertNotifyScalingEngineForEndJob(activeScheduleEntity);
  }

  @Test
  public void testCreateActiveSchedules_when_JobRescheduleMaxCountReached() throws Exception {
    int expectedNumOfTimesJobRescheduled = 5;

    // Build the job
    JobInformation jobInformation =
        new JobInformation<>(AppScalingSpecificDateScheduleStartJob.class);
    Date endJobStartTime = TestDataSetupHelper.getCurrentDateTime(1);
    JobDataMap jobDataMap =
        setupJobDataForSpecificDateSchedule(
            jobInformation.getJobDetail(), endJobStartTime, TimeZone.getDefault());

    ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
    String appId = activeScheduleEntity.getAppId();
    Long scheduleId = activeScheduleEntity.getId();

    embeddedTomcatUtil.addScalingEngineMockForAppAndScheduleId(appId, scheduleId, 200, null);

    Mockito.doThrow(new DatabaseValidationException("test exception"))
        .when(activeScheduleDao)
        .create(any());

    TestJobListener testJobListener = new TestJobListener(expectedNumOfTimesJobRescheduled);
    memScheduler.getListenerManager().addJobListener(testJobListener);

    memScheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

    testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

    // 5 times because in case of failure quartz will reschedule job which will call create again
    Mockito.verify(activeScheduleDao, Mockito.times(expectedNumOfTimesJobRescheduled))
        .deleteActiveSchedulesByAppId(appId);
    Mockito.verify(activeScheduleDao, Mockito.times(expectedNumOfTimesJobRescheduled))
        .create(any());

    // For end job
    Mockito.verify(scheduler, Mockito.never()).scheduleJob(any(), any());

    // For notify to Scaling Engine
    Mockito.verify(restOperations, Mockito.never()).put(Mockito.anyString(), notNull());
  }

  @Test
  public void testRemoveActiveSchedules_when_JobRescheduleMaxCountReached() throws Exception {
    int expectedNumOfTimesJobRescheduled = 5;

    // Build the job
    JobInformation jobInformation = new JobInformation<>(AppScalingScheduleEndJob.class);
    Date endJobStartTime = TestDataSetupHelper.getCurrentDateTime(1);

    long startJobIdentifier = 10L;
    JobDataMap jobDataMap =
        setupJobDataForSpecificDateSchedule(
            jobInformation.getJobDetail(), endJobStartTime, TimeZone.getDefault());
    jobDataMap.put(ScheduleJobHelper.START_JOB_IDENTIFIER, startJobIdentifier);

    ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
    String appId = activeScheduleEntity.getAppId();
    Long scheduleId = activeScheduleEntity.getId();

    embeddedTomcatUtil.addScalingEngineMockForAppAndScheduleId(appId, scheduleId, 204, null);

    Mockito.doThrow(new DatabaseValidationException("test exception"))
        .when(activeScheduleDao)
        .delete(eq(scheduleId), eq(startJobIdentifier));

    TestJobListener testJobListener = new TestJobListener(expectedNumOfTimesJobRescheduled);
    memScheduler.getListenerManager().addJobListener(testJobListener);

    memScheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

    testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

    Mockito.verify(activeScheduleDao, Mockito.times(expectedNumOfTimesJobRescheduled))
        .delete(scheduleId, startJobIdentifier);

    // For notify to Scaling Engine
    Mockito.verify(restOperations, Mockito.never()).delete(Mockito.anyString());
  }

  @Test
  public void
      testNotifyEndOfActiveScheduleToScalingEngine_when_activeScheduleNotFoundInScalingEngine()
          throws Exception {
    // Build the job and trigger
    JobInformation jobInformation = new JobInformation<>(AppScalingScheduleEndJob.class);
    Date endJobStartTime = TestDataSetupHelper.getCurrentDateTime(1);

    long startJobIdentifier = 10L;
    JobDataMap jobDataMap =
        setupJobDataForSpecificDateSchedule(
            jobInformation.getJobDetail(), endJobStartTime, TimeZone.getDefault());
    jobDataMap.put(ScheduleJobHelper.START_JOB_IDENTIFIER, startJobIdentifier);

    ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
    String appId = activeScheduleEntity.getAppId();
    Long scheduleId = activeScheduleEntity.getId();

    embeddedTomcatUtil.addScalingEngineMockForAppAndScheduleId(
        appId, scheduleId, 404, "test not found message");

    TestJobListener testJobListener = new TestJobListener(1);
    memScheduler.getListenerManager().addJobListener(testJobListener);

    memScheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

    testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

    Mockito.verify(activeScheduleDao, Mockito.times(1)).delete(scheduleId, startJobIdentifier);

    // For notify to Scaling Engine
    assertNotifyScalingEngineForEndJob(activeScheduleEntity);
  }

  @Test
  public void testNotifyStartOfActiveScheduleToScalingEngine_when_invalidRequest()
      throws Exception {
    // Build the job and trigger
    JobInformation jobInformation =
        new JobInformation<>(AppScalingSpecificDateScheduleStartJob.class);
    Date endJobStartTime = TestDataSetupHelper.getCurrentDateTime(1);
    JobDataMap jobDataMap =
        setupJobDataForSpecificDateSchedule(
            jobInformation.getJobDetail(), endJobStartTime, TimeZone.getDefault());
    // Min_Count > Max_Count (Invalid data)
    jobDataMap.put(ScheduleJobHelper.INSTANCE_MIN_COUNT, 5);
    jobDataMap.put(ScheduleJobHelper.INSTANCE_MAX_COUNT, 4);

    ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
    String appId = activeScheduleEntity.getAppId();
    Long scheduleId = activeScheduleEntity.getId();

    embeddedTomcatUtil.addScalingEngineMockForAppAndScheduleId(
        appId, scheduleId, 400, "test error message");

    TestJobListener testJobListener = new TestJobListener(1);
    memScheduler.getListenerManager().addJobListener(testJobListener);

    memScheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

    testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

    Mockito.verify(activeScheduleDao, Mockito.times(1)).deleteActiveSchedulesByAppId(appId);
    Mockito.verify(activeScheduleDao, Mockito.times(1)).create(any());

    // For end job
    ArgumentCaptor<JobDetail> jobDetailArgumentCaptor = ArgumentCaptor.forClass(JobDetail.class);
    ArgumentCaptor<Trigger> triggerArgumentCaptor = ArgumentCaptor.forClass(Trigger.class);

    Mockito.verify(scheduler, Mockito.times(1))
        .scheduleJob(jobDetailArgumentCaptor.capture(), triggerArgumentCaptor.capture());

    Long startJobIdentifier =
        jobDetailArgumentCaptor
            .getValue()
            .getJobDataMap()
            .getLong(ScheduleJobHelper.START_JOB_IDENTIFIER);

    assertEndJobArgument(
        triggerArgumentCaptor.getValue(), endJobStartTime, scheduleId, startJobIdentifier);

    // For notify to Scaling Engine
    assertNotifyScalingEngineForStartJob(activeScheduleEntity, startJobIdentifier);
  }

  @Test
  public void testNotifyEndOfActiveScheduleToScalingEngine_when_invalidRequest() throws Exception {
    // Build the job and trigger
    JobInformation jobInformation = new JobInformation<>(AppScalingScheduleEndJob.class);
    Date endJobStartTime = TestDataSetupHelper.getCurrentDateTime(1);

    long startJobIdentifier = 10L;
    JobDataMap jobDataMap =
        setupJobDataForSpecificDateSchedule(
            jobInformation.getJobDetail(), endJobStartTime, TimeZone.getDefault());
    jobDataMap.put(ScheduleJobHelper.START_JOB_IDENTIFIER, startJobIdentifier);
    // Min_Count > Max_Count (Invalid data)
    jobDataMap.put(ScheduleJobHelper.INSTANCE_MIN_COUNT, 5);
    jobDataMap.put(ScheduleJobHelper.INSTANCE_MAX_COUNT, 4);

    ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
    String appId = activeScheduleEntity.getAppId();
    Long scheduleId = activeScheduleEntity.getId();

    embeddedTomcatUtil.addScalingEngineMockForAppAndScheduleId(
        appId, scheduleId, 400, "test error message");

    TestJobListener testJobListener = new TestJobListener(1);
    memScheduler.getListenerManager().addJobListener(testJobListener);

    memScheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

    testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

    Mockito.verify(activeScheduleDao, Mockito.times(1)).delete(scheduleId, startJobIdentifier);

    // For notify to Scaling Engine
    assertNotifyScalingEngineForEndJob(activeScheduleEntity);
  }

  @Test
  public void testNotifyStartOfActiveScheduleToScalingEngine_when_responseError() throws Exception {
    // Build the job and trigger
    JobInformation jobInformation =
        new JobInformation<>(AppScalingSpecificDateScheduleStartJob.class);
    Date endJobStartTime = TestDataSetupHelper.getCurrentDateTime(1);
    JobDataMap jobDataMap =
        setupJobDataForSpecificDateSchedule(
            jobInformation.getJobDetail(), endJobStartTime, TimeZone.getDefault());

    ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
    String appId = activeScheduleEntity.getAppId();
    Long scheduleId = activeScheduleEntity.getId();

    embeddedTomcatUtil.addScalingEngineMockForAppAndScheduleId(
        appId, scheduleId, 500, "test error message");

    TestJobListener testJobListener = new TestJobListener(1);
    memScheduler.getListenerManager().addJobListener(testJobListener);

    memScheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

    testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

    Mockito.verify(activeScheduleDao, Mockito.times(1)).deleteActiveSchedulesByAppId(appId);
    Mockito.verify(activeScheduleDao, Mockito.times(1)).create(any());

    // For end job
    ArgumentCaptor<JobDetail> jobDetailArgumentCaptor = ArgumentCaptor.forClass(JobDetail.class);
    ArgumentCaptor<Trigger> triggerArgumentCaptor = ArgumentCaptor.forClass(Trigger.class);

    Mockito.verify(scheduler, Mockito.times(1))
        .scheduleJob(jobDetailArgumentCaptor.capture(), triggerArgumentCaptor.capture());

    Long startJobIdentifier =
        jobDetailArgumentCaptor
            .getValue()
            .getJobDataMap()
            .getLong(ScheduleJobHelper.START_JOB_IDENTIFIER);

    assertEndJobArgument(
        triggerArgumentCaptor.getValue(), endJobStartTime, scheduleId, startJobIdentifier);

    // For notify to Scaling Engine
    assertNotifyScalingEngineForStartJob(activeScheduleEntity, startJobIdentifier);
  }

  @Test
  public void testNotifyEndOfActiveScheduleToScalingEngine_when_responseError() throws Exception {
    // Build the job and trigger
    JobInformation jobInformation = new JobInformation<>(AppScalingScheduleEndJob.class);
    Date endJobStartTime = TestDataSetupHelper.getCurrentDateTime(1);

    long startJobIdentifier = 10L;
    JobDataMap jobDataMap =
        setupJobDataForSpecificDateSchedule(
            jobInformation.getJobDetail(), endJobStartTime, TimeZone.getDefault());
    jobDataMap.put(ScheduleJobHelper.START_JOB_IDENTIFIER, startJobIdentifier);

    ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
    String appId = activeScheduleEntity.getAppId();
    Long scheduleId = activeScheduleEntity.getId();

    embeddedTomcatUtil.addScalingEngineMockForAppAndScheduleId(
        appId, scheduleId, 500, "test error message");

    TestJobListener testJobListener = new TestJobListener(1);
    memScheduler.getListenerManager().addJobListener(testJobListener);

    memScheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

    testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

    Mockito.verify(activeScheduleDao, Mockito.times(1)).delete(scheduleId, startJobIdentifier);

    // For notify to Scaling Engine
    assertNotifyScalingEngineForEndJob(activeScheduleEntity);
  }

  @Test
  public void testNotifyScalingEngine_when_invalidUrl() throws Exception {
    // Build the job and trigger
    JobInformation jobInformation =
        new JobInformation<>(AppScalingSpecificDateScheduleStartJob.class);
    Date endJobStartTime = TestDataSetupHelper.getCurrentDateTime(1);
    JobDataMap jobDataMap =
        setupJobDataForSpecificDateSchedule(
            jobInformation.getJobDetail(), endJobStartTime, TimeZone.getDefault());

    ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
    String appId = activeScheduleEntity.getAppId();
    Long scheduleId = activeScheduleEntity.getId();

    Mockito.doThrow(new ResourceAccessException("test exception"))
        .when(restOperations)
        .put(eq(scalingEngineUrl + "/v1/apps/" + appId + "/active_schedules/" + scheduleId), any());

    TestJobListener testJobListener = new TestJobListener(2);
    memScheduler.getListenerManager().addJobListener(testJobListener);

    memScheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

    testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

    Mockito.verify(activeScheduleDao, Mockito.times(1)).deleteActiveSchedulesByAppId(appId);
    Mockito.verify(activeScheduleDao, Mockito.times(1)).create(any());

    // For end job
    ArgumentCaptor<JobDetail> jobDetailArgumentCaptor = ArgumentCaptor.forClass(JobDetail.class);
    ArgumentCaptor<Trigger> triggerArgumentCaptor = ArgumentCaptor.forClass(Trigger.class);

    Mockito.verify(scheduler, Mockito.times(1))
        .scheduleJob(jobDetailArgumentCaptor.capture(), triggerArgumentCaptor.capture());

    Long startJobIdentifier =
        jobDetailArgumentCaptor
            .getValue()
            .getJobDataMap()
            .getLong(ScheduleJobHelper.START_JOB_IDENTIFIER);

    assertEndJobArgument(
        triggerArgumentCaptor.getValue(), endJobStartTime, scheduleId, startJobIdentifier);

    // For notify to Scaling Engine
    assertNotifyScalingEngineForStartJob(activeScheduleEntity, startJobIdentifier);
  }

  @Test
  public void testNotifyScalingEngine_when_failed_to_schedule_endJob() throws Exception {
    // Build the job
    JobInformation jobInformation =
        new JobInformation<>(AppScalingSpecificDateScheduleStartJob.class);
    Date endJobStartTime = TestDataSetupHelper.getCurrentDateTime(1);
    JobDataMap jobDataMap =
        setupJobDataForSpecificDateSchedule(
            jobInformation.getJobDetail(), endJobStartTime, TimeZone.getDefault());

    ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
    String appId = activeScheduleEntity.getAppId();
    Long scheduleId = activeScheduleEntity.getId();

    embeddedTomcatUtil.addScalingEngineMockForAppAndScheduleId(appId, scheduleId, 204, null);

    Mockito.doThrow(new SchedulerException("test exception"))
        .when(scheduler)
        .scheduleJob(any(), any());

    TestJobListener testJobListener = new TestJobListener(1);
    memScheduler.getListenerManager().addJobListener(testJobListener);
    memScheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());
    testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

    Mockito.verify(activeScheduleDao, Mockito.times(1)).deleteActiveSchedulesByAppId(appId);
    Mockito.verify(activeScheduleDao, Mockito.times(1)).create(any());

    // For end job
    ArgumentCaptor<JobDetail> jobDetailArgumentCaptor = ArgumentCaptor.forClass(JobDetail.class);
    ArgumentCaptor<Trigger> triggerArgumentCaptor = ArgumentCaptor.forClass(Trigger.class);

    Mockito.verify(scheduler, Mockito.times(1))
        .scheduleJob(jobDetailArgumentCaptor.capture(), triggerArgumentCaptor.capture());

    Long startJobIdentifier =
        jobDetailArgumentCaptor
            .getValue()
            .getJobDataMap()
            .getLong(ScheduleJobHelper.START_JOB_IDENTIFIER);

    assertEndJobArgument(
        triggerArgumentCaptor.getValue(), endJobStartTime, scheduleId, startJobIdentifier);

    // For notify to Scaling Engine
    assertNotifyScalingEngineForStartJob(activeScheduleEntity, startJobIdentifier);
  }

  private void assertEndJobArgument(
      Trigger trigger, Date expectedEndJobStartTime, long scheduleId, long startJobIdentifier) {
    String name = scheduleId + JobActionEnum.END.getJobIdSuffix() + "_" + startJobIdentifier;
    JobKey endJobKey = new JobKey(name, "Schedule");
    TriggerKey endTriggerKey = new TriggerKey(name, "Schedule");
    assertThat(trigger.getJobKey(), is(endJobKey));
    assertThat(trigger.getKey(), is(endTriggerKey));
    assertThat(trigger.getStartTime(), is(expectedEndJobStartTime));
    assertThat(trigger.getMisfireInstruction(), is(SimpleTrigger.MISFIRE_INSTRUCTION_FIRE_NOW));
  }

  private void assertNotifyScalingEngineForStartJob(
      ActiveScheduleEntity activeScheduleEntity, long startJobIdentifier) {
    activeScheduleEntity.setStartJobIdentifier(startJobIdentifier);
    String scalingEnginePath =
        scalingEngineUrl
            + "/v1/apps/"
            + activeScheduleEntity.getAppId()
            + "/active_schedules/"
            + activeScheduleEntity.getId();
    HttpEntity<ActiveScheduleEntity> requestEntity = new HttpEntity<>(activeScheduleEntity);
    Mockito.verify(restOperations, Mockito.times(1)).put(scalingEnginePath, requestEntity);
  }

  private void assertNotifyScalingEngineForEndJob(ActiveScheduleEntity activeScheduleEntity) {
    String scalingEnginePathActiveSchedule =
        scalingEngineUrl
            + "/v1/apps/"
            + activeScheduleEntity.getAppId()
            + "/active_schedules/"
            + activeScheduleEntity.getId();
    Mockito.verify(restOperations, Mockito.times(1)).delete(scalingEnginePathActiveSchedule);
  }

  private JobDataMap setupJobDataForSpecificDateSchedule(
      JobDetail jobDetail, Date startTime, TimeZone timeZone) {
    JobDataMap jobDataMap = TestDataSetupHelper.setupJobDataMap(jobDetail);
    LocalDateTime endJobStartTime =
        LocalDateTime.ofInstant(startTime.toInstant(), timeZone.toZoneId());
    jobDataMap.put(ScheduleJobHelper.END_JOB_START_TIME, endJobStartTime);

    return jobDataMap;
  }

  private JobDataMap setupJobDataForRecurringSchedule(JobDetail jobDetail, String cronExpression) {
    JobDataMap jobDataMap = TestDataSetupHelper.setupJobDataMap(jobDetail);

    jobDataMap.put(ScheduleJobHelper.END_JOB_CRON_EXPRESSION, cronExpression);

    return jobDataMap;
  }
}
