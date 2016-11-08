package org.cloudfoundry.autoscaler.scheduler.rest;

import static org.hamcrest.core.Is.is;
import static org.junit.Assert.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import java.util.Date;
import java.util.TimeZone;
import java.util.concurrent.TimeUnit;

import org.apache.logging.log4j.Level;
import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.core.Appender;
import org.apache.logging.log4j.core.LogEvent;
import org.apache.logging.log4j.core.LoggerContext;
import org.apache.logging.log4j.core.config.Configuration;
import org.apache.logging.log4j.core.config.LoggerConfig;
import org.cloudfoundry.autoscaler.scheduler.entity.ActiveScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationSchedules;
import org.cloudfoundry.autoscaler.scheduler.util.ApplicationPolicyBuilder;
import org.cloudfoundry.autoscaler.scheduler.util.EmbeddedTomcatUtil;
import org.cloudfoundry.autoscaler.scheduler.util.JobActionEnum;
import org.cloudfoundry.autoscaler.scheduler.util.TestConfiguration;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataCleanupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
import org.junit.After;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.ArgumentCaptor;
import org.mockito.Captor;
import org.mockito.Mock;
import org.mockito.Mockito;
import org.quartz.Scheduler;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.context.SpringBootTest.WebEnvironment;
import org.springframework.http.MediaType;
import org.springframework.test.annotation.Commit;
import org.springframework.test.annotation.DirtiesContext;
import org.springframework.test.context.junit4.SpringRunner;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.ResultActions;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.client.RestTemplate;
import org.springframework.web.context.WebApplicationContext;

import com.fasterxml.jackson.databind.ObjectMapper;

@RunWith(SpringRunner.class)
@SpringBootTest(webEnvironment = WebEnvironment.RANDOM_PORT)
@DirtiesContext(classMode = DirtiesContext.ClassMode.BEFORE_CLASS)
@Commit
public class ScheduleRestController_CreateScheduleAndNofifyScalingEngineTest extends TestConfiguration {

	@Mock
	private Appender mockAppender;

	@Captor
	private ArgumentCaptor<LogEvent> logCaptor;

	@Autowired
	private MessageBundleResourceHelper messageBundleResourceHelper;

	@Autowired
	private Scheduler scheduler;

	@Autowired
	private WebApplicationContext wac;
	private MockMvc mockMvc;

	@Autowired
	private TestDataCleanupHelper testDataCleanupHelper;

	@Value("${autoscaler.scalingengine.url}")
	private String scalingEngineUrl;

	@Autowired
	private RestTemplate restTemplate;

	EmbeddedTomcatUtil embeddedTomcatUtil;

	@Before
	@Transactional
	public void before() throws Exception {
		// Clean up data
		testDataCleanupHelper.cleanupData(scheduler);
		embeddedTomcatUtil = new EmbeddedTomcatUtil();
		embeddedTomcatUtil.start();

		Mockito.reset(mockAppender);

		Mockito.when(mockAppender.getName()).thenReturn("MockAppender");
		Mockito.when(mockAppender.isStarted()).thenReturn(true);
		Mockito.when(mockAppender.isStopped()).thenReturn(false);

		mockMvc = MockMvcBuilders.webAppContextSetup(wac).build();

		setLogLevel(Level.INFO);
	}

	@After
	public void after() {
		embeddedTomcatUtil.stop();
	}

	@Test
	public void testCreateScheduleAndNotifyScalingEngine() throws Exception {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];

		Date startTime = new Date(System.currentTimeMillis() + TimeUnit.MINUTES.toMillis(1));
		Date endTime = new Date(System.currentTimeMillis() + TimeUnit.MINUTES.toMillis(2));

		ApplicationSchedules applicationSchedules = new ApplicationPolicyBuilder(1, 5, TimeZone.getDefault().getID(), 1,
				0, 0).build();
		SpecificDateScheduleEntity specificDateScheduleEntity = applicationSchedules.getSchedules().getSpecificDate()
				.get(0);
		specificDateScheduleEntity.setStartDateTime(startTime);
		specificDateScheduleEntity.setEndDateTime(endTime);

		Long currentSequenceSchedulerId = testDataCleanupHelper.getCurrentSequenceSchedulerId() + 1;

		ActiveScheduleEntity startActiveScheduleEntity = new ActiveScheduleEntity();
		startActiveScheduleEntity.setAppId(appId);
		startActiveScheduleEntity.setId(currentSequenceSchedulerId);
		startActiveScheduleEntity.setStatus(JobActionEnum.START.getStatus());
		startActiveScheduleEntity.setInstanceMinCount(specificDateScheduleEntity.getInstanceMinCount());
		startActiveScheduleEntity.setInstanceMaxCount(specificDateScheduleEntity.getInstanceMaxCount());
		startActiveScheduleEntity.setInitialMinInstanceCount(specificDateScheduleEntity.getInitialMinInstanceCount());

		ActiveScheduleEntity endActiveScheduleEntity = new ActiveScheduleEntity();
		endActiveScheduleEntity.setAppId(appId);
		endActiveScheduleEntity.setId(currentSequenceSchedulerId);
		endActiveScheduleEntity.setStatus(JobActionEnum.END.getStatus());
		endActiveScheduleEntity.setInstanceMinCount(applicationSchedules.getInstanceMinCount());
		endActiveScheduleEntity.setInstanceMaxCount(applicationSchedules.getInstanceMaxCount());

		embeddedTomcatUtil.setup(appId, currentSequenceSchedulerId, 200, null);

		//		TestJobListener testStartJobListener = new TestJobListener(JobActionEnum.START.name(),1);
		//		JobKey jobKey = ScheduleJobHelper.generateJobKey(currentSequenceSchedulerId, JobActionEnum.START, ScheduleTypeEnum.SPECIFIC_DATE);
		//		scheduler.getListenerManager().addJobListener(testStartJobListener, KeyMatcher.keyEquals(jobKey));
		//
		//		TestJobListener testEndJobListener = new TestJobListener(JobActionEnum.END.name(),1);
		//		jobKey = ScheduleJobHelper.generateJobKey(currentSequenceSchedulerId, JobActionEnum.END, ScheduleTypeEnum.SPECIFIC_DATE);
		//		scheduler.getListenerManager().addJobListener(testEndJobListener, KeyMatcher.keyEquals(jobKey));

		ObjectMapper mapper = new ObjectMapper();
		String content = mapper.writeValueAsString(applicationSchedules);
		ResultActions resultActions = mockMvc.perform(put(getCreateSchedulerPath(appId))
				.contentType(MediaType.APPLICATION_JSON).accept(MediaType.APPLICATION_JSON).content(content));

		resultActions.andExpect(status().isOk());

		// Assert START Job successful message
		Thread.sleep(TimeUnit.MINUTES.toMillis(1));
		//		testStartJobListener.waitForJobToFinish(TimeUnit.SECONDS.toMillis(75));

		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());
		String expectedMessage = messageBundleResourceHelper.lookupMessage("scalingengine.notification.success",
				startActiveScheduleEntity.getAppId(), startActiveScheduleEntity.getId(), JobActionEnum.START);

		assertThat("Log level should be INFO", logCaptor.getValue().getLevel(), is(Level.INFO));
		assertThat(logCaptor.getValue().getMessage().getFormattedMessage(), is(expectedMessage));

		// Assert END Job successful message
		Thread.sleep(TimeUnit.MINUTES.toMillis(1));
		//		testEndJobListener.waitForJobToFinish(TimeUnit.SECONDS.toMillis(75));

		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());
		expectedMessage = messageBundleResourceHelper.lookupMessage("scalingengine.notification.success",
				endActiveScheduleEntity.getAppId(), endActiveScheduleEntity.getId(), JobActionEnum.END);

		assertThat("Log level should be INFO", logCaptor.getValue().getLevel(), is(Level.INFO));
		assertThat(logCaptor.getValue().getMessage().getFormattedMessage(), is(expectedMessage));

	}

	private String getCreateSchedulerPath(String appId) {
		return String.format("/v2/schedules/%s", appId);
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

}
