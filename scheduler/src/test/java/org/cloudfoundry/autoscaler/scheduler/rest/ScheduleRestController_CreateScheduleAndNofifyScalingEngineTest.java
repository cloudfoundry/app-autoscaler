package org.cloudfoundry.autoscaler.scheduler.rest;

import static org.hamcrest.core.Is.is;
import static org.junit.Assert.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import java.time.LocalDateTime;
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
import org.cloudfoundry.autoscaler.scheduler.util.TestDataDbUtil;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.TestJobListener;
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
import org.quartz.Scheduler;
import org.quartz.impl.matchers.NameMatcher;
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
import org.springframework.test.web.servlet.result.MockMvcResultMatchers;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;
import org.springframework.transaction.annotation.Transactional;
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
	private TestDataDbUtil testDataDbUtil;

	@Value("${autoscaler.scalingengine.url}")
	private String scalingEngineUrl;

	private static EmbeddedTomcatUtil embeddedTomcatUtil;


	@BeforeClass
	public static void beforeClass() {
		embeddedTomcatUtil = new EmbeddedTomcatUtil();
		embeddedTomcatUtil.start();
	}

	@AfterClass
	public static void afterClass() {
		embeddedTomcatUtil.stop();
	}

	@Before
	@Transactional
	public void before() throws Exception {
		// Clean up data
		testDataDbUtil.cleanupData(scheduler);
		Mockito.reset(mockAppender);

		Mockito.when(mockAppender.getName()).thenReturn("MockAppender");
		Mockito.when(mockAppender.isStarted()).thenReturn(true);
		Mockito.when(mockAppender.isStopped()).thenReturn(false);

		mockMvc = MockMvcBuilders.webAppContextSetup(wac).build();

		setLogLevel(Level.INFO);
	}

	@Test
	public void testCreateScheduleAndNotifyScalingEngine() throws Exception {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];

		LocalDateTime startTime = LocalDateTime.now().plusSeconds(70);
		LocalDateTime endTime = LocalDateTime.now().plusSeconds(130);

		ApplicationSchedules applicationSchedules = new ApplicationPolicyBuilder(1, 5, TimeZone.getDefault().getID(), 1,
				0, 0).build();
		SpecificDateScheduleEntity specificDateScheduleEntity = applicationSchedules.getSchedules().getSpecificDate()
				.get(0);
		specificDateScheduleEntity.setStartDateTime(startTime);
		specificDateScheduleEntity.setEndDateTime(endTime);

		ActiveScheduleEntity startActiveScheduleEntity = new ActiveScheduleEntity();
		startActiveScheduleEntity.setAppId(appId);
		startActiveScheduleEntity.setInstanceMinCount(specificDateScheduleEntity.getInstanceMinCount());
		startActiveScheduleEntity.setInstanceMaxCount(specificDateScheduleEntity.getInstanceMaxCount());
		startActiveScheduleEntity.setInitialMinInstanceCount(specificDateScheduleEntity.getInitialMinInstanceCount());

		ActiveScheduleEntity endActiveScheduleEntity = new ActiveScheduleEntity();
		endActiveScheduleEntity.setAppId(appId);
		endActiveScheduleEntity.setInstanceMinCount(applicationSchedules.getInstanceMinCount());
		endActiveScheduleEntity.setInstanceMaxCount(applicationSchedules.getInstanceMaxCount());

		embeddedTomcatUtil.setup(appId, 200, null);
		TestJobListener startJobListener = new TestJobListener(1);
		TestJobListener endJobListener = new TestJobListener(1);

		scheduler.getListenerManager().addJobListener(startJobListener,
				NameMatcher.jobNameEndsWith(JobActionEnum.START.getJobIdSuffix()));
		scheduler.getListenerManager().addJobListener(endJobListener,
				NameMatcher.jobNameContains(JobActionEnum.END.getJobIdSuffix()));

		ObjectMapper mapper = new ObjectMapper();
		String content = mapper.writeValueAsString(applicationSchedules);
		ResultActions resultActions = mockMvc.perform(put(getCreateSchedulerPath(appId))
				.contentType(MediaType.APPLICATION_JSON).accept(MediaType.APPLICATION_JSON).content(content));

		resultActions.andExpect(MockMvcResultMatchers.content().string(""));
		resultActions.andExpect(status().isOk());

		// Assert START Job successful message
		startJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(2));

		Long currentSequenceSchedulerId = testDataDbUtil.getCurrentSequenceSchedulerId();
		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());
		String expectedMessage = messageBundleResourceHelper.lookupMessage(
				"scalingengine.notification.activeschedule.start", startActiveScheduleEntity.getAppId(),
				currentSequenceSchedulerId);

		assertThat(logCaptor.getValue().getMessage().getFormattedMessage(), is(expectedMessage));
		assertThat("Log level should be INFO", logCaptor.getValue().getLevel(), is(Level.INFO));

		// Assert END Job successful message
		endJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(2));

		Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());
		expectedMessage = messageBundleResourceHelper.lookupMessage("scalingengine.notification.activeschedule.remove",
				endActiveScheduleEntity.getAppId(), currentSequenceSchedulerId);

		assertThat(logCaptor.getValue().getMessage().getFormattedMessage(), is(expectedMessage));
		assertThat("Log level should be INFO", logCaptor.getValue().getLevel(), is(Level.INFO));

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
