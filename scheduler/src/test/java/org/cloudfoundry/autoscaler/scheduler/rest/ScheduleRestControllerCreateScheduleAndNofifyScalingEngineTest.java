package org.cloudfoundry.autoscaler.scheduler.rest;

import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.collection.IsEmptyCollection.empty;
import static org.hamcrest.core.Is.is;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.delete;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import com.fasterxml.jackson.databind.ObjectMapper;
import java.io.IOException;
import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.List;
import java.util.TimeZone;
import java.util.concurrent.TimeUnit;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationSchedules;
import org.cloudfoundry.autoscaler.scheduler.util.ApplicationPolicyBuilder;
import org.cloudfoundry.autoscaler.scheduler.util.EmbeddedTomcatUtil;
import org.cloudfoundry.autoscaler.scheduler.util.JobActionEnum;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataDbUtil;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.TestJobListener;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
import org.hamcrest.Matchers;
import org.junit.AfterClass;
import org.junit.Before;
import org.junit.BeforeClass;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.quartz.JobKey;
import org.quartz.Scheduler;
import org.quartz.SchedulerException;
import org.quartz.impl.matchers.GroupMatcher;
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

@RunWith(SpringRunner.class)
@SpringBootTest(webEnvironment = WebEnvironment.RANDOM_PORT)
@DirtiesContext(classMode = DirtiesContext.ClassMode.BEFORE_CLASS)
@Commit
public class ScheduleRestControllerCreateScheduleAndNofifyScalingEngineTest {

  @Autowired private MessageBundleResourceHelper messageBundleResourceHelper;

  @Autowired private Scheduler scheduler;

  @Autowired private WebApplicationContext wac;
  private MockMvc mockMvc;

  @Autowired private TestDataDbUtil testDataDbUtil;

  @Value("${autoscaler.scalingengine.url}")
  private String scalingEngineUrl;

  private static EmbeddedTomcatUtil embeddedTomcatUtil;

  @BeforeClass
  public static void beforeClass() throws IOException {
    embeddedTomcatUtil = new EmbeddedTomcatUtil();
    embeddedTomcatUtil.start();
  }

  @AfterClass
  public static void afterClass() throws IOException, InterruptedException {
    embeddedTomcatUtil.stop();
  }

  private String appId;
  private String guid;

  private TestJobListener startJobListener;

  private TestJobListener endJobListener;

  @Before
  @Transactional
  public void before() throws Exception {
    // Clean up data
    testDataDbUtil.cleanupData(scheduler);

    mockMvc = MockMvcBuilders.webAppContextSetup(wac).build();

    appId = TestDataSetupHelper.generateAppIds(1)[0];
    guid = TestDataSetupHelper.generateGuid();
    startJobListener = new TestJobListener(1);
    endJobListener = new TestJobListener(1);
  }

  @Test
  public void testCreateScheduleAndNotifyScalingEngine() throws Exception {
    createSchedule();

    // Assert START Job successful message
    startJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(2));

    Long currentSequenceSchedulerId = testDataDbUtil.getCurrentSpecificDateSchedulerId();

    // Assert END Job successful message
    endJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(2));

    assertThat(
        "It should have no active schedule",
        testDataDbUtil.getNumberOfActiveSchedulesByAppId(appId),
        Matchers.is(0L));
  }

  @Test
  public void testDeleteSchedule() throws Exception {
    createSchedule();

    // Assert START Job successful message
    startJobListener.waitForJobToFinish(TimeUnit.MINUTES.toMillis(2));

    Long currentSequenceSchedulerId = testDataDbUtil.getCurrentSpecificDateSchedulerId();

    // Delete End job.
    ResultActions resultActions =
        mockMvc.perform(
            delete(TestDataSetupHelper.getSchedulerPath(appId)).accept(MediaType.APPLICATION_JSON));

    resultActions.andExpect(MockMvcResultMatchers.content().string(""));
    resultActions.andExpect(status().isNoContent());

    // Assert END Job doesn't exist
    assertThat("It should not have any job keys.", getExistingJobKeys(), empty());
  }

  @Test
  public void testQuartzSetting() throws SchedulerException {
    assertThat(scheduler.getSchedulerName(), is("app-autoscaler"));
    assertThat(scheduler.getSchedulerInstanceId(), is("scheduler-12345"));
  }

  public void createSchedule() throws Exception {
    LocalDateTime startTime = LocalDateTime.now().plusSeconds(70);
    LocalDateTime endTime = LocalDateTime.now().plusSeconds(130);

    ApplicationSchedules applicationSchedules =
        new ApplicationPolicyBuilder(1, 5, TimeZone.getDefault().getID(), 1, 0, 0).build();
    SpecificDateScheduleEntity specificDateScheduleEntity =
        applicationSchedules.getSchedules().getSpecificDate().get(0);
    specificDateScheduleEntity.setStartDateTime(startTime);
    specificDateScheduleEntity.setEndDateTime(endTime);

    embeddedTomcatUtil.addScalingEngineMockForAppId(appId, 200, null);

    scheduler
        .getListenerManager()
        .addJobListener(
            startJobListener, NameMatcher.jobNameEndsWith(JobActionEnum.START.getJobIdSuffix()));
    scheduler
        .getListenerManager()
        .addJobListener(
            endJobListener, NameMatcher.jobNameContains(JobActionEnum.END.getJobIdSuffix()));

    ObjectMapper mapper = new ObjectMapper();
    String content = mapper.writeValueAsString(applicationSchedules);
    ResultActions resultActions =
        mockMvc.perform(
            put(TestDataSetupHelper.getSchedulerPath(appId))
                .param("guid", guid)
                .contentType(MediaType.APPLICATION_JSON)
                .accept(MediaType.APPLICATION_JSON)
                .content(content));

    resultActions.andExpect(MockMvcResultMatchers.content().string(""));
    resultActions.andExpect(status().isOk());
  }

  private List<JobKey> getExistingJobKeys() throws SchedulerException {
    List<JobKey> jobKeys = new ArrayList<>();

    for (JobKey jobkey : scheduler.getJobKeys(GroupMatcher.anyGroup())) {
      jobKeys.add(jobkey);
    }

    return jobKeys;
  }
}
