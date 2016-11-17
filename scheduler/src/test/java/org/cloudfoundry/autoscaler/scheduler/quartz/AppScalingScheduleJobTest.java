package org.cloudfoundry.autoscaler.scheduler.quartz;

import static org.hamcrest.core.Is.is;
import static org.junit.Assert.assertNotNull;
import static org.junit.Assert.assertThat;
import static org.mockito.Matchers.eq;

import java.net.URL;
import java.util.Date;
import java.util.List;
import java.util.concurrent.TimeUnit;

import org.apache.logging.log4j.Level;
import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.core.Appender;
import org.apache.logging.log4j.core.LogEvent;
import org.apache.logging.log4j.core.LoggerContext;
import org.apache.logging.log4j.core.config.Configuration;
import org.apache.logging.log4j.core.config.LoggerConfig;
import org.cloudfoundry.autoscaler.scheduler.dao.ActiveScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.ActiveScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.EmbeddedTomcatUtil;
import org.cloudfoundry.autoscaler.scheduler.util.JobActionEnum;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleTypeEnum;
import org.cloudfoundry.autoscaler.scheduler.util.TestConfiguration;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataCleanupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.TestJobListener;
import org.cloudfoundry.autoscaler.scheduler.util.error.DatabaseValidationException;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
import org.junit.AfterClass;
import org.junit.Before;
import org.junit.BeforeClass;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.ArgumentCaptor;
import org.mockito.Captor;
import org.mockito.Mock;
import org.mockito.Mockito;
import org.mockito.MockitoAnnotations;
import org.quartz.JobBuilder;
import org.quartz.JobDataMap;
import org.quartz.JobDetail;
import org.quartz.JobKey;
import org.quartz.Scheduler;
import org.quartz.SchedulerException;
import org.quartz.Trigger;
import org.quartz.TriggerKey;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.mock.mockito.SpyBean;
import org.springframework.http.HttpEntity;
import org.springframework.test.context.junit4.SpringRunner;
import org.springframework.web.client.ResourceAccessException;
import org.springframework.web.client.RestTemplate;

@RunWith(SpringRunner.class)
@SpringBootTest
public class AppScalingScheduleJobTest extends TestConfiguration {

	@Mock
	private Appender mockAppender;

	@Captor
	private ArgumentCaptor<LogEvent> logCaptor;

	@Autowired
	private MessageBundleResourceHelper messageBundleResourceHelper;

	@Autowired
	private Scheduler scheduler;

	@SpyBean
	private ActiveScheduleDao activeScheduleDao;

	@SpyBean
	private RestTemplate restTemplate;

	@Autowired
	private TestDataCleanupHelper testDataCleanupHelper;

	@Value("${autoscaler.scalingengine.url}")
	private String scalingEngineUrl;

	static EmbeddedTomcatUtil embeddedTomcatUtil;

	@BeforeClass
	public static void beforClass() {
		embeddedTomcatUtil = new EmbeddedTomcatUtil();
		embeddedTomcatUtil.start();

	}

	@AfterClass
	public static void afterClass() {
		embeddedTomcatUtil.stop();
	}

	@Before
	public void before() throws SchedulerException {
		MockitoAnnotations.initMocks(this);
		testDataCleanupHelper.cleanupData(scheduler);
		Mockito.reset(mockAppender);

		Mockito.when(mockAppender.getName()).thenReturn("MockAppender");
		Mockito.when(mockAppender.isStarted()).thenReturn(true);
		Mockito.when(mockAppender.isStopped()).thenReturn(false);

		setLogLevel(Level.INFO);
	}

	@Test
	public void testNotifyStartOfActiveScheduleToScalingEngine() throws Exception {
		// Build the job and trigger
		JobInformation jobInformation = new JobInformation<>(AppScalingScheduleStartJob.class);
		JobDataMap jobDataMap = setupJobData(jobInformation.getJobDetail(), JobActionEnum.START);

		ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
		String appId = activeScheduleEntity.getAppId();
		Long scheduleId = activeScheduleEntity.getId();

		embeddedTomcatUtil.setup(appId, scheduleId, 200, null);

		ArgumentCaptor<ActiveScheduleEntity> activeScheduleEntityArgumentCaptor = ArgumentCaptor
				.forClass(ActiveScheduleEntity.class);

		TestJobListener testJobListener = new TestJobListener(1);
		scheduler.getListenerManager().addJobListener(testJobListener);

		scheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

		testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

		assertNotNull(activeScheduleDao);
		assertNotNull(activeScheduleEntityArgumentCaptor);

		Mockito.verify(activeScheduleDao, Mockito.atLeastOnce()).create(activeScheduleEntity);
		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());

		String expectedMessage = messageBundleResourceHelper.lookupMessage("scalingengine.notification.activeschedule.start", appId,
				scheduleId, JobActionEnum.START);
		assertThat("Log level should be INFO", logCaptor.getValue().getLevel(), is(Level.INFO));
		assertThat(logCaptor.getValue().getMessage().getFormattedMessage(), is(expectedMessage));

	}

	@Test
	public void testNotifyEndOfActiveScheduleToScalingEngine() throws Exception {
		// Build the job
		JobInformation jobInformation = new JobInformation<>(AppScalingScheduleEndJob.class);
		JobDataMap jobDataMap = setupJobData(jobInformation.getJobDetail(), JobActionEnum.END);

		ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
		String appId = activeScheduleEntity.getAppId();
		Long scheduleId = activeScheduleEntity.getId();

		embeddedTomcatUtil.setup(appId, scheduleId, 204, null);

		ArgumentCaptor<URL> urlArgumentCaptor = ArgumentCaptor.forClass(URL.class);
		ArgumentCaptor<ActiveScheduleEntity> activeScheduleEntityArgumentCaptor = ArgumentCaptor
				.forClass(ActiveScheduleEntity.class);

		TestJobListener testJobListener = new TestJobListener(1);
		scheduler.getListenerManager().addJobListener(testJobListener);
		scheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());
		testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

		Mockito.verify(activeScheduleDao, Mockito.times(1)).delete(activeScheduleEntity.getId());
		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());

		String expectedMessage = messageBundleResourceHelper.lookupMessage("scalingengine.notification.activeschedule.remove", appId,
				scheduleId, JobActionEnum.END);
		assertThat("Log level should be INFO", logCaptor.getValue().getLevel(), is(Level.INFO));
		assertThat(logCaptor.getValue().getMessage().getFormattedMessage(), is(expectedMessage));

	}

	@Test
	public void testCreateActiveSchedules_throw_DatabaseValidationException() throws Exception {
		setLogLevel(Level.ERROR);

		int expectedNumOfJobFired = 2;

		// Build the job
		JobInformation jobInformation = new JobInformation<>(AppScalingScheduleStartJob.class);
		JobDataMap jobDataMap = setupJobData(jobInformation.getJobDetail(), JobActionEnum.START);

		ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
		String appId = activeScheduleEntity.getAppId();
		Long scheduleId = activeScheduleEntity.getId();

		embeddedTomcatUtil.setup(appId, scheduleId, 200, null);

		Mockito.doThrow(new DatabaseValidationException("test exception")).doNothing().when(activeScheduleDao)
				.create(Mockito.anyObject());

		TestJobListener testJobListener = new TestJobListener(expectedNumOfJobFired);
		scheduler.getListenerManager().addJobListener(testJobListener);

		scheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

		testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

		Mockito.verify(activeScheduleDao, Mockito.times(expectedNumOfJobFired)).create(activeScheduleEntity);

		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());
		String expectedMessage = messageBundleResourceHelper
				.lookupMessage("database.error.create.activeschedule.failed", "test exception", appId, scheduleId);

		assertThat("Log level should be ERROR", logCaptor.getValue().getLevel(), is(Level.ERROR));
		assertThat(logCaptor.getValue().getMessage().getFormattedMessage(), is(expectedMessage));

	}

	@Test
	public void testRemoveActiveSchedules_throw_DatabaseValidationException() throws Exception {
		setLogLevel(Level.ERROR);

		int expectedNumOfJobFired = 2;

		// Build the job
		JobInformation jobInformation = new JobInformation<>(AppScalingScheduleEndJob.class);
		JobDataMap jobDataMap = setupJobData(jobInformation.getJobDetail(), JobActionEnum.END);

		ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
		String appId = activeScheduleEntity.getAppId();
		Long scheduleId = activeScheduleEntity.getId();

		embeddedTomcatUtil.setup(appId, scheduleId, 204, null);

		Mockito.doThrow(new DatabaseValidationException("test exception")).doReturn(1).when(activeScheduleDao)
				.delete(scheduleId);

		TestJobListener testJobListener = new TestJobListener(expectedNumOfJobFired);
		scheduler.getListenerManager().addJobListener(testJobListener);

		scheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

		testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

		Mockito.verify(activeScheduleDao, Mockito.times(expectedNumOfJobFired)).delete(scheduleId);
		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());
		String expectedMessage = messageBundleResourceHelper
				.lookupMessage("database.error.delete.activeschedule.failed", "test exception", appId, scheduleId);

		assertThat("Log level should be ERROR", logCaptor.getValue().getLevel(), is(Level.ERROR));
		assertThat(logCaptor.getValue().getMessage().getFormattedMessage(), is(expectedMessage));
	}

	@Test
	//@Ignore
	public void testCreateActiveSchedules_when_MaxRescheduleCountReached() throws Exception {
		setLogLevel(Level.ERROR);

		int expectedNumOfJobFired = 5;

		// Build the job
		JobInformation jobInformation = new JobInformation<>(AppScalingScheduleStartJob.class);
		JobDataMap jobDataMap = setupJobData(jobInformation.getJobDetail(), JobActionEnum.START);

		ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
		String appId = activeScheduleEntity.getAppId();
		Long scheduleId = activeScheduleEntity.getId();

		embeddedTomcatUtil.setup(appId, scheduleId, 200, null);

		Mockito.doThrow(new DatabaseValidationException("test exception")).when(activeScheduleDao)
				.create(Mockito.anyObject());

		TestJobListener testJobListener = new TestJobListener(expectedNumOfJobFired);
		scheduler.getListenerManager().addJobListener(testJobListener);

		scheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

		testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

		// 5 times because in case of failure quartz will reschedule job which will call create again
		Mockito.verify(activeScheduleDao, Mockito.times(expectedNumOfJobFired)).create(activeScheduleEntity);

		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());
		String expectedMessage = messageBundleResourceHelper.lookupMessage(
				"scheduler.job.reschedule.failed.max.reached", jobInformation.getTrigger().getKey(), appId, scheduleId,
				expectedNumOfJobFired, ScheduleJobHelper.RescheduleCount.ACTIVE_SCHEDULE.name());

		assertThat("Log level should be ERROR", logCaptor.getValue().getLevel(), is(Level.ERROR));
		assertThat(logCaptor.getValue().getMessage().getFormattedMessage(), is(expectedMessage));

	}

	@Test
	public void testRemoveActiveSchedules_when_MaxRescheduleCountReached() throws Exception {
		setLogLevel(Level.ERROR);

		int expectedNumOfJobFired = 5;

		// Build the job
		JobInformation jobInformation = new JobInformation<>(AppScalingScheduleEndJob.class);
		JobDataMap jobDataMap = setupJobData(jobInformation.getJobDetail(), JobActionEnum.END);

		ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
		String appId = activeScheduleEntity.getAppId();
		Long scheduleId = activeScheduleEntity.getId();

		embeddedTomcatUtil.setup(appId, scheduleId, 204, null);

		Mockito.doThrow(new DatabaseValidationException("test exception")).when(activeScheduleDao).delete(scheduleId);

		TestJobListener testJobListener = new TestJobListener(expectedNumOfJobFired);
		scheduler.getListenerManager().addJobListener(testJobListener);

		scheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

		testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

		Mockito.verify(activeScheduleDao, Mockito.times(expectedNumOfJobFired)).delete(scheduleId);
		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());
		String expectedMessage = messageBundleResourceHelper.lookupMessage(
				"scheduler.job.reschedule.failed.max.reached", jobInformation.getTrigger().getKey(), appId, scheduleId,
				expectedNumOfJobFired, ScheduleJobHelper.RescheduleCount.ACTIVE_SCHEDULE.name());

		assertThat("Log level should be ERROR", logCaptor.getValue().getLevel(), is(Level.ERROR));
		assertThat(logCaptor.getValue().getMessage().getFormattedMessage(), is(expectedMessage));
	}

	@Test
	public void testNotifyStartOfActiveScheduleToScalingEngine_when_invalidRequest() throws Exception {
		setLogLevel(Level.ERROR);
		// Build the job and trigger
		JobInformation jobInformation = new JobInformation<>(AppScalingScheduleStartJob.class);
		JobDataMap jobDataMap = setupJobData(jobInformation.getJobDetail(), JobActionEnum.START);
		// Min_Count > Max_Count (Invalid data)
		jobDataMap.put(ScheduleJobHelper.INSTANCE_MIN_COUNT, 5);
		jobDataMap.put(ScheduleJobHelper.INSTANCE_MAX_COUNT, 4);

		ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
		String appId = activeScheduleEntity.getAppId();
		Long scheduleId = activeScheduleEntity.getId();

		embeddedTomcatUtil.setup(appId, scheduleId, 400, "test error message");

		TestJobListener testJobListener = new TestJobListener(1);
		scheduler.getListenerManager().addJobListener(testJobListener);

		scheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

		testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

		Mockito.verify(activeScheduleDao, Mockito.times(1)).create(activeScheduleEntity);

		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());
		String expectedMessage = messageBundleResourceHelper.lookupMessage("scalingengine.notification.client.error",
				400, "test error message", appId, scheduleId, JobActionEnum.START);

		assertThat("Log level should be ERROR", logCaptor.getValue().getLevel(), is(Level.ERROR));
		assertThat(logCaptor.getValue().getMessage().getFormattedMessage(), is(expectedMessage));
	}

	@Test
	public void testNotifyEndOfActiveScheduleToScalingEngine_when_invalidRequest() throws Exception {
		setLogLevel(Level.ERROR);
		// Build the job and trigger
		JobInformation jobInformation = new JobInformation<>(AppScalingScheduleEndJob.class);
		JobDataMap jobDataMap = setupJobData(jobInformation.getJobDetail(), JobActionEnum.END);
		// Min_Count > Max_Count (Invalid data)
		jobDataMap.put(ScheduleJobHelper.INSTANCE_MIN_COUNT, 5);
		jobDataMap.put(ScheduleJobHelper.INSTANCE_MAX_COUNT, 4);

		ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
		String appId = activeScheduleEntity.getAppId();
		Long scheduleId = activeScheduleEntity.getId();

		embeddedTomcatUtil.setup(appId, scheduleId, 400, "test error message");

		TestJobListener testJobListener = new TestJobListener(1);
		scheduler.getListenerManager().addJobListener(testJobListener);

		scheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

		testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

		Mockito.verify(activeScheduleDao, Mockito.times(1)).delete(scheduleId);

		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());
		String expectedMessage = messageBundleResourceHelper.lookupMessage("scalingengine.notification.client.error",
				400, "test error message", appId, scheduleId, JobActionEnum.END);

		assertThat("Log level should be ERROR", logCaptor.getValue().getLevel(), is(Level.ERROR));
		assertThat(logCaptor.getValue().getMessage().getFormattedMessage(), is(expectedMessage));
	}

	@Test
	public void testNotifyStartOfActiveScheduleToScalingEngine_when_responseError() throws Exception {
		setLogLevel(Level.ERROR);
		// Build the job and trigger
		JobInformation jobInformation = new JobInformation<>(AppScalingScheduleStartJob.class);
		JobDataMap jobDataMap = setupJobData(jobInformation.getJobDetail(), JobActionEnum.START);

		ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
		String appId = activeScheduleEntity.getAppId();
		Long scheduleId = activeScheduleEntity.getId();

		embeddedTomcatUtil.setup(appId, scheduleId, 500, "test error message");

		TestJobListener testJobListener = new TestJobListener(1);
		scheduler.getListenerManager().addJobListener(testJobListener);

		scheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

		testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

		Mockito.verify(activeScheduleDao, Mockito.times(1)).create(activeScheduleEntity);
		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());
		String expectedMessage = messageBundleResourceHelper.lookupMessage("scalingengine.notification.failed", 500,
				"test error message", appId, scheduleId, JobActionEnum.START);

		assertThat("Log level should be ERROR", logCaptor.getValue().getLevel(), is(Level.ERROR));
		assertThat(logCaptor.getValue().getMessage().getFormattedMessage(), is(expectedMessage));

	}

	@Test
	public void testNotifyEndOfActiveScheduleToScalingEngine_when_responseError() throws Exception {
		setLogLevel(Level.ERROR);
		// Build the job and trigger
		JobInformation jobInformation = new JobInformation<>(AppScalingScheduleEndJob.class);
		JobDataMap jobDataMap = setupJobData(jobInformation.getJobDetail(), JobActionEnum.END);

		ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
		String appId = activeScheduleEntity.getAppId();
		Long scheduleId = activeScheduleEntity.getId();

		embeddedTomcatUtil.setup(appId, scheduleId, 500, "test error message");

		TestJobListener testJobListener = new TestJobListener(1);
		scheduler.getListenerManager().addJobListener(testJobListener);

		scheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

		testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

		Mockito.verify(activeScheduleDao, Mockito.times(1)).delete(scheduleId);
		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());
		String expectedMessage = messageBundleResourceHelper.lookupMessage("scalingengine.notification.failed", 500,
				"test error message", appId, scheduleId, JobActionEnum.END);

		assertThat("Log level should be ERROR", logCaptor.getValue().getLevel(), is(Level.ERROR));
		assertThat(logCaptor.getValue().getMessage().getFormattedMessage(), is(expectedMessage));

	}

	@Test
	public void testNotifyScalingEngine_when_invalidURL() throws Exception {
		setLogLevel(Level.ERROR);

		// Build the job and trigger
		JobInformation jobInformation = new JobInformation<>(AppScalingScheduleStartJob.class);
		JobDataMap jobDataMap = setupJobData(jobInformation.getJobDetail(), JobActionEnum.START);

		ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
		String appId = activeScheduleEntity.getAppId();
		Long scheduleId = activeScheduleEntity.getId();

		HttpEntity<ActiveScheduleEntity> requestEntity = new HttpEntity<>(activeScheduleEntity);
		Mockito.doThrow(new ResourceAccessException("test exception")).when(restTemplate)
				.put(eq(scalingEngineUrl + "/v1/apps/" + appId + "/active_schedules/" + scheduleId), eq(requestEntity));

		TestJobListener testJobListener = new TestJobListener(2);
		scheduler.getListenerManager().addJobListener(testJobListener);

		scheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

		testJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(1));

		Mockito.verify(activeScheduleDao, Mockito.times(1)).create(activeScheduleEntity);
		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());
		String expectedMessage = messageBundleResourceHelper.lookupMessage("scalingengine.notification.error",
				"test exception", appId, scheduleId, JobActionEnum.START);

		AssertLogHasMessageCount(Level.ERROR, expectedMessage, 2);

		expectedMessage = messageBundleResourceHelper.lookupMessage("scheduler.job.reschedule.failed.max.reached",
				jobInformation.getTrigger().getKey(), appId, scheduleId, 2,
				ScheduleJobHelper.RescheduleCount.SCALING_ENGINE_NOTIFICATION.name());

		assertThat("Log level should be ERROR", logCaptor.getValue().getLevel(), is(Level.ERROR));
		assertThat(logCaptor.getValue().getMessage().getFormattedMessage(), is(expectedMessage));

	}

	private void AssertLogHasMessageCount(Level logLevel, String expectedMessage, int expectedCount) {
		int messageCount = 0;
		List<LogEvent> logEvents = logCaptor.getAllValues();
		for (LogEvent logEvent : logEvents) {
			if (logEvent.getLevel() == logLevel
					&& logEvent.getMessage().getFormattedMessage().equals(expectedMessage)) {
				++messageCount;
			}
		}
		assertThat("Log should have message", messageCount, is(expectedCount));
	}

	private void setLogLevel(Level level) {
		LoggerContext ctx = (LoggerContext) LogManager.getContext(false);
		Configuration config = ctx.getConfiguration();

		LoggerConfig loggerConfig = config.getLoggerConfig(LogManager.ROOT_LOGGER_NAME);
		loggerConfig.removeAppender("MockAppender");

		loggerConfig.setLevel(level);
		loggerConfig.addAppender(mockAppender, level, null);
		ctx.updateLoggers();

	}

	private JobDataMap setupJobData(JobDetail jobDetail, JobActionEnum jobAction) {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		Long scheduleId = 1L;

		JobDataMap jobDataMap = jobDetail.getJobDataMap();
		jobDataMap.put(ScheduleJobHelper.APP_ID, appId);
		jobDataMap.put(ScheduleJobHelper.SCHEDULE_ID, scheduleId);
		jobDataMap.put(ScheduleJobHelper.INITIAL_MIN_INSTANCE_COUNT, 1);
		jobDataMap.put(ScheduleJobHelper.INSTANCE_MIN_COUNT, 2);
		jobDataMap.put(ScheduleJobHelper.INSTANCE_MAX_COUNT, 4);
		jobDataMap.put(ScheduleJobHelper.RescheduleCount.ACTIVE_SCHEDULE.name(), 1);
		jobDataMap.put(ScheduleJobHelper.RescheduleCount.SCALING_ENGINE_NOTIFICATION.name(), 1);
		jobDataMap.put(ScheduleJobHelper.ACTIVE_SCHEDULE_TABLE_TASK_DONE, false);

		return jobDataMap;
	}

	private static class JobInformation<T extends AppScalingScheduleJob> {
		private JobDetail jobDetail;
		private Trigger trigger;

		JobInformation(Class<T> appScalingScheduleJobClass) {
			JobKey jobKey = new JobKey("TestJobKey", ScheduleTypeEnum.SPECIFIC_DATE.getScheduleIdentifier());
			this.jobDetail = JobBuilder.newJob(appScalingScheduleJobClass).withIdentity(jobKey).storeDurably().build();

			Date triggerTime = new Date(System.currentTimeMillis() + TimeUnit.SECONDS.toMillis(1));
			TriggerKey triggerKey = new TriggerKey("TestTriggerKey",
					ScheduleTypeEnum.SPECIFIC_DATE.getScheduleIdentifier());

			this.trigger = ScheduleJobHelper.buildTrigger(triggerKey, jobKey, triggerTime);
		}

		JobDetail getJobDetail() {
			return jobDetail;
		}

		Trigger getTrigger() {
			return trigger;
		}
	}
}