package org.cloudfoundry.autoscaler.scheduler.quartz;

import static org.hamcrest.core.Is.is;
import static org.junit.Assert.assertThat;

import java.io.IOException;
import java.net.HttpURLConnection;
import java.net.URL;
import java.util.Date;
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
import org.cloudfoundry.autoscaler.scheduler.util.JobActionEnum;
import org.cloudfoundry.autoscaler.scheduler.util.ScalingEngineUtil;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleTypeEnum;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataCleanupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.DatabaseValidationException;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
import org.junit.AfterClass;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.ArgumentCaptor;
import org.mockito.Captor;
import org.mockito.Mock;
import org.mockito.Mockito;
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
import org.springframework.test.context.ActiveProfiles;
import org.springframework.test.context.junit4.SpringRunner;

@ActiveProfiles(value = { "ActiveScheduleDaoMock", "ScalingEngineUtilMock" })
@RunWith(SpringRunner.class)
@SpringBootTest
public class AppScalingScheduleJobTest {

	@Mock
	private Appender mockAppender;

	@Captor
	private ArgumentCaptor<LogEvent> logCaptor;

	@Autowired
	private MessageBundleResourceHelper messageBundleResourceHelper;

	@Autowired
	private Scheduler scheduler;

	@Autowired
	private ActiveScheduleDao activeScheduleDao;

	@Autowired
	private TestDataCleanupHelper testDataCleanupHelper;

	@Value("${autoscaler.scalingengine.url}")
	private String scalingEngineUrl;

	@Autowired
	private ScalingEngineUtil scalingEngineUtil;

	@Value("${scalingenginejob.refire.interval}")
	private static int originalJobRefireInterval;

	@Value("${scalingenginejob.refire.maxcount}")
	private static long originalJobRefireMaxCount;

	@Before
	public void before() throws SchedulerException {

		testDataCleanupHelper.cleanupData(scheduler);

		Mockito.reset(activeScheduleDao);
		Mockito.reset(scalingEngineUtil);
		Mockito.reset(mockAppender);

		Mockito.when(mockAppender.getName()).thenReturn("MockAppender");
		Mockito.when(mockAppender.isStarted()).thenReturn(true);
		Mockito.when(mockAppender.isStopped()).thenReturn(false);

		setLogLevel(Level.INFO);
	}

	@AfterClass
	public static void afterClass() {
		System.setProperty("scalingenginejob.refire.interval", String.valueOf(originalJobRefireInterval));
		System.setProperty("scalingenginejob.refire.maxcount", String.valueOf(originalJobRefireMaxCount));
	}

	@Test
	public void testNotifyStartOfActiveScheduleToScalingEngine() throws Exception {
		// Build the job and trigger
		JobInformation jobInformation = new JobInformation<>(AppScalingScheduleStartJob.class);

		JobDataMap jobDataMap = setupJobData(jobInformation.getJobDetail(), JobActionEnum.START);
		ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);

		HttpURLConnection mockHttpURLConnection = Mockito.mock(HttpURLConnection.class);
		ArgumentCaptor<URL> urlArgumentCaptor = ArgumentCaptor.forClass(URL.class);
		ArgumentCaptor<ActiveScheduleEntity> activeScheduleEntityArgumentCaptor = ArgumentCaptor
				.forClass(ActiveScheduleEntity.class);
		Mockito.when(scalingEngineUtil.getConnection(urlArgumentCaptor.capture(),
				activeScheduleEntityArgumentCaptor.capture())).thenReturn(mockHttpURLConnection);

		scheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

		//sleep
		Thread.sleep(TimeUnit.SECONDS.toMillis(2));

		Mockito.verify(activeScheduleDao, Mockito.times(1)).create(activeScheduleEntity);

		URL url = new URL(scalingEngineUrl + "/v1/apps/" + activeScheduleEntity.getAppId() + "/active_schedule/"
				+ activeScheduleEntity.getId());
		assertThat("It should be equal", urlArgumentCaptor.getValue(), is(url));

		assertThat("It should be equal", activeScheduleEntityArgumentCaptor.getValue(), is(activeScheduleEntity));

		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());

		String expectedMessage = messageBundleResourceHelper.lookupMessage("scalingengine.notification.success",
				activeScheduleEntity.getAppId(), activeScheduleEntity.getId(), JobActionEnum.START);
		assertThat("Log level should be INFO", logCaptor.getValue().getLevel(), is(Level.INFO));
		assertThat(logCaptor.getValue().getMessage().getFormattedMessage(), is(expectedMessage));

	}

	@Test
	public void testNotifyEndOfActiveScheduleToScalingEngine() throws Exception {
		// Build the job
		JobInformation jobInformation = new JobInformation<>(AppScalingScheduleEndJob.class);

		JobDataMap jobDataMap = setupJobData(jobInformation.getJobDetail(), JobActionEnum.END);
		ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);

		HttpURLConnection mockHttpURLConnection = Mockito.mock(HttpURLConnection.class);
		ArgumentCaptor<URL> urlArgumentCaptor = ArgumentCaptor.forClass(URL.class);
		ArgumentCaptor<ActiveScheduleEntity> activeScheduleEntityArgumentCaptor = ArgumentCaptor
				.forClass(ActiveScheduleEntity.class);
		Mockito.when(scalingEngineUtil.getConnection(urlArgumentCaptor.capture(),
				activeScheduleEntityArgumentCaptor.capture())).thenReturn(mockHttpURLConnection);

		scheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

		//sleep
		Thread.sleep(TimeUnit.SECONDS.toMillis(2));

		Mockito.verify(activeScheduleDao, Mockito.times(1)).delete(activeScheduleEntity.getId());

		URL url = new URL(scalingEngineUrl + "/v1/apps/" + activeScheduleEntity.getAppId() + "/active_schedule/"
				+ activeScheduleEntity.getId());
		assertThat("It should be equal", urlArgumentCaptor.getValue(), is(url));

		assertThat("It should be equal", activeScheduleEntityArgumentCaptor.getValue(), is(activeScheduleEntity));

		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());

		String expectedMessage = messageBundleResourceHelper.lookupMessage("scalingengine.notification.success",
				activeScheduleEntity.getAppId(), activeScheduleEntity.getId(), JobActionEnum.END);
		assertThat("Log level should be INFO", logCaptor.getValue().getLevel(), is(Level.INFO));
		assertThat(logCaptor.getValue().getMessage().getFormattedMessage(), is(expectedMessage));

	}

	@Test
	public void testCreateActiveSchedules_throw_DatabaseValidationException() throws Exception {
		setLogLevel(Level.ERROR);
		System.setProperty("scalingenginejob.refire.interval", String.valueOf(100));
		System.setProperty("scalingenginejob.refire.maxcount", String.valueOf(4));

		// Build the job
		JobInformation jobInformation = new JobInformation<>(AppScalingScheduleStartJob.class);

		JobDataMap jobDataMap = setupJobData(jobInformation.getJobDetail(), JobActionEnum.START);
		ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);

		Mockito.doThrow(new DatabaseValidationException("test exception")).when(activeScheduleDao)
				.create(Mockito.anyObject());

		HttpURLConnection mockHttpURLConnection = Mockito.mock(HttpURLConnection.class);
		Mockito.when(scalingEngineUtil.getConnection(Mockito.anyObject(), Mockito.anyObject()))
				.thenReturn(mockHttpURLConnection);

		scheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

		//sleep
		Thread.sleep(TimeUnit.SECONDS.toMillis(3));

		// 2 times because in case of failure quartz will refire job immediately which will call create again
		Mockito.verify(activeScheduleDao, Mockito.times(5)).create(activeScheduleEntity);

		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());
		String expectedMessage = messageBundleResourceHelper
				.lookupMessage("database.error.create.activeschedule.failed", "test exception");

		assertThat("Log level should be ERROR", logCaptor.getValue().getLevel(), is(Level.ERROR));
		assertThat(logCaptor.getValue().getMessage().getFormattedMessage(), is(expectedMessage));

	}

	@Test
	public void testRemoveActiveSchedules_throw_DatabaseValidationException() throws Exception {
		setLogLevel(Level.ERROR);

		System.setProperty("scalingenginejob.refire.interval", String.valueOf(100));
		System.setProperty("scalingenginejob.refire.maxcount", String.valueOf(3));

		// Build the job
		JobInformation jobInformation = new JobInformation<>(AppScalingScheduleEndJob.class);
		JobDataMap jobDataMap = setupJobData(jobInformation.getJobDetail(), JobActionEnum.END);
		ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);

		Mockito.doThrow(new DatabaseValidationException("test exception")).when(activeScheduleDao)
				.delete(activeScheduleEntity.getId());

		HttpURLConnection mockHttpURLConnection = Mockito.mock(HttpURLConnection.class);
		Mockito.when(scalingEngineUtil.getConnection(Mockito.anyObject(), Mockito.anyObject()))
				.thenReturn(mockHttpURLConnection);

		scheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

		//sleep
		Thread.sleep(TimeUnit.SECONDS.toMillis(3));

		Mockito.verify(activeScheduleDao, Mockito.times(4)).delete(activeScheduleEntity.getId());
		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());
		String expectedMessage = messageBundleResourceHelper
				.lookupMessage("database.error.delete.activeschedule.failed", "test exception");

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

		HttpURLConnection mockHttpURLConnection = Mockito.mock(HttpURLConnection.class);
		Mockito.when(scalingEngineUtil.getConnection(Mockito.anyObject(), Mockito.anyObject()))
				.thenReturn(mockHttpURLConnection);
		Mockito.when(mockHttpURLConnection.getResponseCode()).thenReturn(400);
		Mockito.when(mockHttpURLConnection.getResponseMessage()).thenReturn("test error message");

		scheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());
		//sleep
		Thread.sleep(TimeUnit.SECONDS.toMillis(2));

		Mockito.verify(activeScheduleDao, Mockito.times(1)).create(activeScheduleEntity);

		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());
		String expectedMessage = messageBundleResourceHelper.lookupMessage("scalingengine.notification.client.error",
				400, "test error message", activeScheduleEntity.getAppId(), activeScheduleEntity.getId(),
				JobActionEnum.START);

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

		HttpURLConnection mockHttpURLConnection = Mockito.mock(HttpURLConnection.class);
		Mockito.when(scalingEngineUtil.getConnection(Mockito.anyObject(), Mockito.anyObject()))
				.thenReturn(mockHttpURLConnection);
		Mockito.when(mockHttpURLConnection.getResponseCode()).thenReturn(400);
		Mockito.when(mockHttpURLConnection.getResponseMessage()).thenReturn("test error message");

		scheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

		//sleep
		Thread.sleep(TimeUnit.SECONDS.toMillis(2));

		Mockito.verify(activeScheduleDao, Mockito.times(1)).delete(activeScheduleEntity.getId());

		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());
		String expectedMessage = messageBundleResourceHelper.lookupMessage("scalingengine.notification.client.error",
				400, "test error message", activeScheduleEntity.getAppId(), activeScheduleEntity.getId(),
				JobActionEnum.END);

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

		HttpURLConnection mockHttpURLConnection = Mockito.mock(HttpURLConnection.class);
		Mockito.when(scalingEngineUtil.getConnection(Mockito.anyObject(), Mockito.anyObject()))
				.thenReturn(mockHttpURLConnection);
		Mockito.when(mockHttpURLConnection.getResponseCode()).thenReturn(500);
		Mockito.when(mockHttpURLConnection.getResponseMessage()).thenReturn("test error message");

		scheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

		//sleep
		Thread.sleep(TimeUnit.SECONDS.toMillis(2));

		Mockito.verify(activeScheduleDao, Mockito.times(1)).create(activeScheduleEntity);
		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());
		String expectedMessage = messageBundleResourceHelper.lookupMessage("scalingengine.notification.failed", 500,
				"test error message", activeScheduleEntity.getAppId(), activeScheduleEntity.getId(),
				JobActionEnum.START);

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

		HttpURLConnection mockHttpURLConnection = Mockito.mock(HttpURLConnection.class);
		Mockito.when(scalingEngineUtil.getConnection(Mockito.anyObject(), Mockito.anyObject()))
				.thenReturn(mockHttpURLConnection);
		Mockito.when(mockHttpURLConnection.getResponseCode()).thenReturn(500);
		Mockito.when(mockHttpURLConnection.getResponseMessage()).thenReturn("test error message");

		scheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

		//sleep
		Thread.sleep(TimeUnit.SECONDS.toMillis(2));

		Mockito.verify(activeScheduleDao, Mockito.times(1)).delete(activeScheduleEntity.getId());
		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());
		String expectedMessage = messageBundleResourceHelper.lookupMessage("scalingengine.notification.failed", 500,
				"test error message", activeScheduleEntity.getAppId(), activeScheduleEntity.getId(), JobActionEnum.END);

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

		HttpURLConnection mockHttpURLConnection = Mockito.mock(HttpURLConnection.class);
		Mockito.when(scalingEngineUtil.getConnection(Mockito.anyObject(), Mockito.anyObject()))
				.thenReturn(mockHttpURLConnection);
		Mockito.when(mockHttpURLConnection.getResponseCode()).thenReturn(404);
		Mockito.when(mockHttpURLConnection.getResponseMessage()).thenReturn("test error message");

		scheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

		//sleep
		Thread.sleep(TimeUnit.SECONDS.toMillis(2));

		Mockito.verify(activeScheduleDao, Mockito.times(1)).create(activeScheduleEntity);
		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());
		String expectedMessage = messageBundleResourceHelper.lookupMessage("scalingengine.notification.client.error",
				404, "test error message", activeScheduleEntity.getAppId(), activeScheduleEntity.getId(),
				JobActionEnum.START);

		assertThat("Log level should be ERROR", logCaptor.getValue().getLevel(), is(Level.ERROR));
		assertThat(logCaptor.getValue().getMessage().getFormattedMessage(), is(expectedMessage));

	}

	@Test
	public void testNotifyScalingEngine_throw_IOException() throws Exception {
		setLogLevel(Level.ERROR);

		// Build the job and trigger
		JobInformation jobInformation = new JobInformation<>(AppScalingScheduleStartJob.class);

		JobDataMap jobDataMap = setupJobData(jobInformation.getJobDetail(), JobActionEnum.START);
		ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);

		Mockito.when(scalingEngineUtil.getConnection(Mockito.anyObject(), Mockito.anyObject()))
				.thenThrow(new IOException("test error message"));

		scheduler.scheduleJob(jobInformation.getJobDetail(), jobInformation.getTrigger());

		//sleep
		Thread.sleep(TimeUnit.SECONDS.toMillis(2));

		Mockito.verify(activeScheduleDao, Mockito.times(1)).create(activeScheduleEntity);
		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());
		String expectedMessage = messageBundleResourceHelper.lookupMessage("scalingengine.notification.error",
				"test error message", activeScheduleEntity.getAppId(), activeScheduleEntity.getId(),
				JobActionEnum.START);

		assertThat("Log level should be ERROR", logCaptor.getValue().getLevel(), is(Level.ERROR));
		assertThat(logCaptor.getValue().getMessage().getFormattedMessage(), is(expectedMessage));

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
		jobDataMap.put(ScheduleJobHelper.SCALING_ACTION, jobAction.getStatus());
		jobDataMap.put(ScheduleJobHelper.INITIAL_MIN_INSTANCE_COUNT, 1);
		jobDataMap.put(ScheduleJobHelper.INSTANCE_MIN_COUNT, 2);
		jobDataMap.put(ScheduleJobHelper.INSTANCE_MAX_COUNT, 4);
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
