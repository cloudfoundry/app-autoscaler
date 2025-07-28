package org.cloudfoundry.autoscaler.scheduler.rest;

import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.Matchers.is;
import static org.mockito.ArgumentMatchers.any;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.cloudfoundry.autoscaler.scheduler.dao.ActiveScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.ActiveScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.Schedules;
import org.cloudfoundry.autoscaler.scheduler.rest.model.SynchronizeResult;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataDbUtil;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.Mockito;
import org.quartz.Scheduler;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.context.SpringBootTest.WebEnvironment;
import org.springframework.test.annotation.DirtiesContext;
import org.springframework.test.annotation.DirtiesContext.ClassMode;
import org.springframework.test.context.bean.override.mockito.MockitoBean;
import org.springframework.test.context.junit4.SpringRunner;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.ResultActions;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;
import org.springframework.web.context.WebApplicationContext;

@RunWith(SpringRunner.class)
@SpringBootTest(webEnvironment = WebEnvironment.RANDOM_PORT)
@DirtiesContext(classMode = ClassMode.BEFORE_CLASS)
public class ScheduleSyncRestControllerTest {

  @MockitoBean private Scheduler scheduler;

  @MockitoBean private ActiveScheduleDao activeScheduleDao;

  @Autowired private TestDataDbUtil testDataDbUtil;

  @Autowired private WebApplicationContext wac;

  private MockMvc mockMvc;

  @Before
  public void before() throws Exception {
    Mockito.reset(scheduler);
    Mockito.reset(activeScheduleDao);
    testDataDbUtil.cleanupData();
    mockMvc = MockMvcBuilders.webAppContextSetup(wac).build();

    ActiveScheduleEntity activeScheduleEntity = new ActiveScheduleEntity();
    activeScheduleEntity.setStartJobIdentifier(System.currentTimeMillis());
    Mockito.when(activeScheduleDao.find(any())).thenReturn(activeScheduleEntity);
  }

  @Test
  public void
      testSynchronizeSchedules_with_both_policy_and_schedules_existed_and_guid_are_different()
          throws Exception {

    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String guid = TestDataSetupHelper.generateGuid();
    String anotherGuid = TestDataSetupHelper.generateGuid();
    int noOfSpecificDateSchedules = 3;
    int noOfDomRecurringSchedules = 3;
    int noOfDowRecurringSchedules = 3;

    Schedules schedules =
        TestDataSetupHelper.generateSchedulesWithEntitiesOnly(
            appId,
            guid,
            false,
            noOfSpecificDateSchedules,
            noOfDomRecurringSchedules,
            noOfDowRecurringSchedules);

    testDataDbUtil.insertRecurringSchedule(schedules.getRecurringSchedule());
    testDataDbUtil.insertSpecificDateSchedule(schedules.getSpecificDate());
    testDataDbUtil.insertPolicyJson(appId, anotherGuid);

    ResultActions resultActions = mockMvc.perform(put("/v1/syncSchedules"));
    resultActions.andExpect(status().isOk());
    SynchronizeResult result =
        new ObjectMapper()
            .readValue(
                resultActions.andReturn().getResponse().getContentAsString(),
                SynchronizeResult.class);
    assertThat("It should update the shedules", result, is(new SynchronizeResult(0, 1, 0)));
  }

  @Test
  public void
      testSynchronizeSchedules_with_both_policy_and_schedules_existed_and_guid_are_the_same()
          throws Exception {

    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String guid = TestDataSetupHelper.generateGuid();
    int noOfSpecificDateSchedules = 3;
    int noOfDomRecurringSchedules = 3;
    int noOfDowRecurringSchedules = 3;

    Schedules schedules =
        TestDataSetupHelper.generateSchedulesWithEntitiesOnly(
            appId,
            guid,
            false,
            noOfSpecificDateSchedules,
            noOfDomRecurringSchedules,
            noOfDowRecurringSchedules);

    testDataDbUtil.insertRecurringSchedule(schedules.getRecurringSchedule());
    testDataDbUtil.insertSpecificDateSchedule(schedules.getSpecificDate());
    testDataDbUtil.insertPolicyJson(appId, guid);

    ResultActions resultActions = mockMvc.perform(put("/v1/syncSchedules"));
    resultActions.andExpect(status().isOk());
    SynchronizeResult result =
        new ObjectMapper()
            .readValue(
                resultActions.andReturn().getResponse().getContentAsString(),
                SynchronizeResult.class);
    assertThat(
        "It should not update or create schedule", result, is(new SynchronizeResult(0, 0, 0)));
  }

  @Test
  public void testSynchronizeSchedules_with_no_policy_and_existed_schedules() throws Exception {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String guid = TestDataSetupHelper.generateGuid();
    int noOfSpecificDateSchedules = 3;
    int noOfDomRecurringSchedules = 3;
    int noOfDowRecurringSchedules = 3;

    Schedules schedules =
        TestDataSetupHelper.generateSchedulesWithEntitiesOnly(
            appId,
            guid,
            false,
            noOfSpecificDateSchedules,
            noOfDomRecurringSchedules,
            noOfDowRecurringSchedules);

    testDataDbUtil.insertRecurringSchedule(schedules.getRecurringSchedule());
    testDataDbUtil.insertSpecificDateSchedule(schedules.getSpecificDate());

    ResultActions resultActions = mockMvc.perform(put("/v1/syncSchedules"));
    resultActions.andExpect(status().isOk());
    SynchronizeResult result =
        new ObjectMapper()
            .readValue(
                resultActions.andReturn().getResponse().getContentAsString(),
                SynchronizeResult.class);
    assertThat("It should delete the schedules", result, is(new SynchronizeResult(0, 0, 1)));
  }

  @Test
  public void testSynchronizeSchedules_with_existed_policy_and_no_schedule() throws Exception {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String anotherGuid = TestDataSetupHelper.generateGuid();
    testDataDbUtil.insertPolicyJson(appId, anotherGuid);

    ResultActions resultActions = mockMvc.perform(put("/v1/syncSchedules"));
    resultActions.andExpect(status().isOk());
    SynchronizeResult result =
        new ObjectMapper()
            .readValue(
                resultActions.andReturn().getResponse().getContentAsString(),
                SynchronizeResult.class);
    assertThat("It should create schedules", result, is(new SynchronizeResult(1, 0, 0)));
  }

  @Test
  public void testSynchronizeSchedules_with_no_policy_and_no_schedules() throws Exception {

    ResultActions resultActions = mockMvc.perform(put("/v1/syncSchedules"));
    resultActions.andExpect(status().isOk());
    SynchronizeResult result =
        new ObjectMapper()
            .readValue(
                resultActions.andReturn().getResponse().getContentAsString(),
                SynchronizeResult.class);
    assertThat("It should do nothing", result, is(new SynchronizeResult(0, 0, 0)));
  }
}
