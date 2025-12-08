package org.cloudfoundry.autoscaler.scheduler.rest;

import static org.hamcrest.CoreMatchers.is;
import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.Matchers.emptyString;
import static org.junit.Assert.assertEquals;
import static org.mockito.ArgumentMatchers.any;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.delete;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.content;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.header;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.io.IOException;
import java.text.DateFormat;
import java.util.List;
import org.cloudfoundry.autoscaler.scheduler.dao.ActiveScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.dao.SpecificDateScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.ActiveScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationSchedules;
import org.cloudfoundry.autoscaler.scheduler.util.PolicyUtil;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataDbUtil;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
import org.hamcrest.Matchers;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.Mockito;
import org.quartz.Scheduler;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.context.SpringBootTest.WebEnvironment;
import org.springframework.http.MediaType;
import org.springframework.test.annotation.DirtiesContext;
import org.springframework.test.annotation.DirtiesContext.ClassMode;
import org.springframework.test.context.bean.override.mockito.MockitoBean;
import org.springframework.test.context.junit4.SpringRunner;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.ResultActions;
import org.springframework.test.web.servlet.ResultMatcher;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;
import org.springframework.web.context.WebApplicationContext;

@RunWith(SpringRunner.class)
@SpringBootTest(webEnvironment = WebEnvironment.RANDOM_PORT)
@DirtiesContext(classMode = ClassMode.BEFORE_CLASS)
public class ScheduleRestControllerTest {

  @MockitoBean private Scheduler scheduler;

  @MockitoBean private ActiveScheduleDao activeScheduleDao;

  @Autowired private SpecificDateScheduleDao specificDateScheduleDao;

  @Autowired private MessageBundleResourceHelper messageBundleResourceHelper;

  @Autowired private TestDataDbUtil testDataDbUtil;

  @Autowired private WebApplicationContext wac;

  private MockMvc mockMvc;

  private String guid3;

  String appIdForAppWithOneSpecificDateSchedule;
  String appIdForAppWithOneRecurringSchedule;
  String appIdForAppWithTwoSpecificAndTwowRecuringSchedules;

  @Before
  public void before() throws Exception {
    Mockito.reset(scheduler);
    Mockito.reset(activeScheduleDao);
    testDataDbUtil.cleanupData();
    mockMvc = MockMvcBuilders.webAppContextSetup(wac).build();

    ActiveScheduleEntity activeScheduleEntity = new ActiveScheduleEntity();
    activeScheduleEntity.setStartJobIdentifier(System.currentTimeMillis());
    Mockito.when(activeScheduleDao.find(any())).thenReturn(activeScheduleEntity);

    appIdForAppWithOneSpecificDateSchedule = TestDataSetupHelper.generateGuid();
    String guid1 = TestDataSetupHelper.generateGuid();
    List<SpecificDateScheduleEntity> specificDateScheduleEntities =
        TestDataSetupHelper.generateSpecificDateScheduleEntities(
            appIdForAppWithOneSpecificDateSchedule, guid1, false, 1);
    testDataDbUtil.insertSpecificDateSchedule(specificDateScheduleEntities);

    appIdForAppWithOneRecurringSchedule = TestDataSetupHelper.generateGuid();
    String guid2 = TestDataSetupHelper.generateGuid();
    List<RecurringScheduleEntity> recurringScheduleEntities =
        TestDataSetupHelper.generateRecurringScheduleEntities(
            appIdForAppWithOneRecurringSchedule, guid2, false, 1, 0);
    testDataDbUtil.insertRecurringSchedule(recurringScheduleEntities);

    appIdForAppWithTwoSpecificAndTwowRecuringSchedules = TestDataSetupHelper.generateGuid();
    guid3 = TestDataSetupHelper.generateGuid();
    specificDateScheduleEntities =
        TestDataSetupHelper.generateSpecificDateScheduleEntities(
            appIdForAppWithTwoSpecificAndTwowRecuringSchedules, guid3, false, 2);
    testDataDbUtil.insertSpecificDateSchedule(specificDateScheduleEntities);
    recurringScheduleEntities =
        TestDataSetupHelper.generateRecurringScheduleEntities(
            appIdForAppWithTwoSpecificAndTwowRecuringSchedules, guid3, false, 1, 2);
    testDataDbUtil.insertRecurringSchedule(recurringScheduleEntities);
  }

  @Test
  public void testGetAllSchedule_with_malformed_appId() throws Exception {
    String appId = "appId_1";

    ResultActions resultActions =
        mockMvc.perform(
            get(TestDataSetupHelper.getSchedulerPath(appId)).accept(MediaType.APPLICATION_JSON));

    resultActions.andExpect(status().isBadRequest());
  }

  @Test
  public void testGetAllSchedule_with_no_schedules() throws Exception {
    ResultActions resultActions =
        mockMvc.perform(
            get(TestDataSetupHelper.getSchedulerPath(TestDataSetupHelper.generateGuid()))
                .accept(MediaType.APPLICATION_JSON));

    assertNoSchedulesFound(resultActions);
  }

  @Test
  public void testGetSchedule_with_only_specificDateSchedule() throws Exception {
    ResultActions resultActions =
        mockMvc.perform(
            get(TestDataSetupHelper.getSchedulerPath(appIdForAppWithOneSpecificDateSchedule))
                .accept(MediaType.APPLICATION_JSON));

    ApplicationSchedules applicationPolicy =
        getApplicationSchedulesFromResultActions(resultActions);

    resultActions.andExpect(status().isOk());
    resultActions.andExpect(content().contentTypeCompatibleWith(MediaType.APPLICATION_JSON));

    assertSpecificDateScheduleFoundEquals(
        1,
        appIdForAppWithOneSpecificDateSchedule,
        applicationPolicy.getSchedules().getSpecificDate());
    assertRecurringDateScheduleFoundEquals(
        0,
        appIdForAppWithOneSpecificDateSchedule,
        applicationPolicy.getSchedules().getRecurringSchedule());
  }

  @Test
  public void testGetSchedule_with_only_recurringSchedule() throws Exception {
    ResultActions resultActions =
        mockMvc.perform(
            get(TestDataSetupHelper.getSchedulerPath(appIdForAppWithOneRecurringSchedule))
                .accept(MediaType.APPLICATION_JSON));

    ApplicationSchedules applicationPolicy =
        getApplicationSchedulesFromResultActions(resultActions);

    resultActions.andExpect(status().isOk());
    resultActions.andExpect(content().contentTypeCompatibleWith(MediaType.APPLICATION_JSON));

    assertSpecificDateScheduleFoundEquals(
        0, appIdForAppWithOneRecurringSchedule, applicationPolicy.getSchedules().getSpecificDate());
    assertRecurringDateScheduleFoundEquals(
        1,
        appIdForAppWithOneRecurringSchedule,
        applicationPolicy.getSchedules().getRecurringSchedule());
  }

  @Test
  public void testGetSchedule_with_specificDateSchedule_and_recurringSchedule() throws Exception {
    ResultActions resultActions =
        mockMvc.perform(
            get(TestDataSetupHelper.getSchedulerPath(
                    appIdForAppWithTwoSpecificAndTwowRecuringSchedules))
                .accept(MediaType.APPLICATION_JSON));

    ApplicationSchedules applicationPolicy =
        getApplicationSchedulesFromResultActions(resultActions);

    resultActions.andExpect(status().isOk());
    resultActions.andExpect(content().contentTypeCompatibleWith(MediaType.APPLICATION_JSON));

    assertSpecificDateScheduleFoundEquals(
        2,
        appIdForAppWithTwoSpecificAndTwowRecuringSchedules,
        applicationPolicy.getSchedules().getSpecificDate());
    assertRecurringDateScheduleFoundEquals(
        3,
        appIdForAppWithTwoSpecificAndTwowRecuringSchedules,
        applicationPolicy.getSchedules().getRecurringSchedule());
  }

  @Test
  public void testCreateAndGetSchedules_from_jsonFile() throws Exception {
    String policyJsonStr = PolicyUtil.getPolicyJsonContent();
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String guid = TestDataSetupHelper.generateGuid();
    ResultActions resultActions =
        mockMvc.perform(
            put(TestDataSetupHelper.getSchedulerPath(appId))
                .param("guid", guid)
                .contentType(MediaType.APPLICATION_JSON)
                .accept(MediaType.APPLICATION_JSON)
                .content(policyJsonStr));
    assertResponseForCreateSchedules(resultActions, status().isOk());

    resultActions =
        mockMvc.perform(
            get(TestDataSetupHelper.getSchedulerPath(appId)).accept(MediaType.APPLICATION_JSON));

    ApplicationSchedules applicationSchedules =
        getApplicationSchedulesFromResultActions(resultActions);
    assertSchedulesFoundEquals(applicationSchedules, appId, resultActions, 4, 3);

    Mockito.verify(scheduler, Mockito.times(7)).scheduleJob(any(), any());
  }

  @Test
  public void testCreateSchedule_with_only_specificDateSchedules() throws Exception {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String guid = TestDataSetupHelper.generateGuid();
    String content = TestDataSetupHelper.generateJsonSchedule(2, 0);

    ResultActions resultActions =
        mockMvc.perform(
            put(TestDataSetupHelper.getSchedulerPath(appId))
                .param("guid", guid)
                .contentType(MediaType.APPLICATION_JSON)
                .accept(MediaType.APPLICATION_JSON)
                .content(content));

    assertResponseForCreateSchedules(resultActions, status().isOk());

    assertThat(
        "It should have two specific date schedules.",
        testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(appId),
        is(2));
    assertThat(
        "It should have no recurring schedules.",
        testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId),
        is(0));

    Mockito.verify(scheduler, Mockito.times(2)).scheduleJob(any(), any());
  }

  @Test
  public void testCreateSchedule_with_only_recurringSchedules() throws Exception {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String guid = TestDataSetupHelper.generateGuid();
    String content = TestDataSetupHelper.generateJsonSchedule(0, 2);

    ResultActions resultActions =
        mockMvc.perform(
            put(TestDataSetupHelper.getSchedulerPath(appId))
                .param("guid", guid)
                .contentType(MediaType.APPLICATION_JSON)
                .accept(MediaType.APPLICATION_JSON)
                .content(content));

    assertResponseForCreateSchedules(resultActions, status().isOk());

    assertThat(
        "It should have no specific date schedules.",
        testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(appId),
        is(0));
    assertThat(
        "It should have two recurring schedules.",
        testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId),
        is(2));

    Mockito.verify(scheduler, Mockito.times(2)).scheduleJob(any(), any());
  }

  @Test
  public void testCreateSchedule_with_specificDateSchedules_and_recurringSchedules()
      throws Exception {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String guid = TestDataSetupHelper.generateGuid();
    String content = TestDataSetupHelper.generateJsonSchedule(2, 2);

    ResultActions resultActions =
        mockMvc.perform(
            put(TestDataSetupHelper.getSchedulerPath(appId))
                .param("guid", guid)
                .contentType(MediaType.APPLICATION_JSON)
                .accept(MediaType.APPLICATION_JSON)
                .content(content));

    assertResponseForCreateSchedules(resultActions, status().isOk());

    assertThat(
        "It should have two specific date schedules.",
        testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(appId),
        is(2));
    assertThat(
        "It should have two recurring schedules.",
        testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId),
        is(2));

    Mockito.verify(scheduler, Mockito.times(4)).scheduleJob(any(), any());
  }

  @Test
  public void testCreateSchedule_when_schedule_existing_for_appId() throws Exception {
    assertThat(
        "It should have 2 specific date schedule.",
        testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(
            appIdForAppWithTwoSpecificAndTwowRecuringSchedules),
        is(2));
    assertThat(
        "It should have 3 recurring schedules.",
        testDataDbUtil.getNumberOfRecurringSchedulesByAppId(
            appIdForAppWithTwoSpecificAndTwowRecuringSchedules),
        is(3));

    // Create two specific date schedules and one recurring schedule for the same application.
    String content = TestDataSetupHelper.generateJsonSchedule(2, 1);
    ResultActions resultActions =
        mockMvc.perform(
            put(TestDataSetupHelper.getSchedulerPath(
                    appIdForAppWithTwoSpecificAndTwowRecuringSchedules))
                .param("guid", guid3)
                .contentType(MediaType.APPLICATION_JSON)
                .accept(MediaType.APPLICATION_JSON)
                .content(content));
    assertResponseForCreateSchedules(resultActions, status().isNoContent());

    assertThat(
        "It should have 2 specific date schedules.",
        testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(
            appIdForAppWithTwoSpecificAndTwowRecuringSchedules),
        is(2));
    assertThat(
        "It should have 1 recurring schedule.",
        testDataDbUtil.getNumberOfRecurringSchedulesByAppId(
            appIdForAppWithTwoSpecificAndTwowRecuringSchedules),
        is(1));

    Mockito.verify(scheduler, Mockito.times(3)).scheduleJob(any(), any());
    Mockito.verify(scheduler, Mockito.times(10)).deleteJob(any());
  }

  @Test
  public void testCreateSchedule_without_appId() throws Exception {

    String content = getPolicyString();

    ResultActions resultActions =
        mockMvc.perform(
            put("/v1/apps/schedules").contentType(MediaType.APPLICATION_JSON).content(content));

    resultActions.andExpect(status().isNotFound());
  }

  @Test
  public void testCreateSchedule_with_malformed_appId() throws Exception {

    String appId = "appId";
    String content = getPolicyString();

    ResultActions resultActions =
        mockMvc.perform(
            put(TestDataSetupHelper.getSchedulerPath(appId))
                .contentType(MediaType.APPLICATION_JSON)
                .content(content));

    resultActions.andExpect(status().isBadRequest());
  }

  private static String getPolicyString() throws JsonProcessingException {
    ObjectMapper mapper = new ObjectMapper();
    ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);
    String content = mapper.writeValueAsString(applicationPolicy);
    return content;
  }

  @Test
  public void testCreateSchedule_without_guid() throws Exception {

    String appId = "appId";
    String content = getPolicyString();

    ResultActions resultActions =
        mockMvc.perform(
            put(TestDataSetupHelper.getSchedulerPath(appId))
                .contentType(MediaType.APPLICATION_JSON)
                .content(content));

    resultActions.andExpect(status().isBadRequest());
  }

  @Test
  public void testDeleteSchedule_with_only_specificDateSchedule() throws Exception {
    assertThat(
        "It should have 1 specific date schedule.",
        testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(
            appIdForAppWithOneSpecificDateSchedule),
        is(1));
    assertThat(
        "It should have no recurring schedules.",
        testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appIdForAppWithOneSpecificDateSchedule),
        is(0));

    ResultActions resultActions =
        mockMvc.perform(
            delete(TestDataSetupHelper.getSchedulerPath(appIdForAppWithOneSpecificDateSchedule))
                .accept(MediaType.APPLICATION_JSON));
    assertSchedulesAreDeleted(resultActions);

    assertThat(
        "It should have no specific date schedules.",
        testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(
            appIdForAppWithOneSpecificDateSchedule),
        is(0));
    assertThat(
        "It should have no recurring schedules.",
        testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appIdForAppWithOneSpecificDateSchedule),
        is(0));

    Mockito.verify(scheduler, Mockito.times(2)).deleteJob(any());
  }

  @Test
  public void testDeleteSchedule_with_only_recurringSchedule() throws Exception {
    assertThat(
        "It should have no specific date schedules.",
        testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(appIdForAppWithOneRecurringSchedule),
        is(0));
    assertThat(
        "It should have 1 recurring schedule.",
        testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appIdForAppWithOneRecurringSchedule),
        is(1));

    ResultActions resultActions =
        mockMvc.perform(
            delete(TestDataSetupHelper.getSchedulerPath(appIdForAppWithOneRecurringSchedule))
                .accept(MediaType.APPLICATION_JSON));
    assertSchedulesAreDeleted(resultActions);

    assertThat(
        "It should have no specific date schedules.",
        testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(appIdForAppWithOneRecurringSchedule),
        is(0));
    assertThat(
        "It should have no recurring schedules.",
        testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appIdForAppWithOneRecurringSchedule),
        is(0));

    Mockito.verify(scheduler, Mockito.times(2)).deleteJob(any());
  }

  @Test
  public void testDeleteSchedule_with_specificDateSchedule_and_recurringSchedule()
      throws Exception {
    String appId = "appId_3";

    assertThat(
        "It should have 2 specific date schedule.",
        testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(
            appIdForAppWithTwoSpecificAndTwowRecuringSchedules),
        is(2));
    assertThat(
        "It should have 3 recurring schedule.",
        testDataDbUtil.getNumberOfRecurringSchedulesByAppId(
            appIdForAppWithTwoSpecificAndTwowRecuringSchedules),
        is(3));

    ResultActions resultActions =
        mockMvc.perform(
            delete(
                    TestDataSetupHelper.getSchedulerPath(
                        appIdForAppWithTwoSpecificAndTwowRecuringSchedules))
                .accept(MediaType.APPLICATION_JSON));
    assertSchedulesAreDeleted(resultActions);

    assertThat(
        "It should have no specific date schedules.",
        testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(
            appIdForAppWithTwoSpecificAndTwowRecuringSchedules),
        is(0));
    assertThat(
        "It should have no recurring schedules.",
        testDataDbUtil.getNumberOfRecurringSchedulesByAppId(
            appIdForAppWithTwoSpecificAndTwowRecuringSchedules),
        is(0));

    Mockito.verify(scheduler, Mockito.times(10)).deleteJob(any());
  }

  @Test
  public void testDeleteSchedules_appId_without_schedules() throws Exception {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];

    assertThat(
        "It should have no specific date schedules.",
        testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(appId),
        is(0));
    assertThat(
        "It should have no recurring schedules.",
        testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId),
        is(0));

    assertThat(
        "It should have 3 specific date schedules.",
        testDataDbUtil.getNumberOfSpecificDateSchedules(),
        is(3));
    assertThat(
        "It should have 4 recurring schedules.",
        testDataDbUtil.getNumberOfRecurringSchedules(),
        is(4));

    ResultActions resultActions =
        mockMvc.perform(
            delete(TestDataSetupHelper.getSchedulerPath(appId)).accept(MediaType.APPLICATION_JSON));
    assertNoSchedulesFound(resultActions);

    assertThat(
        "It should have 3 specific date schedules.",
        testDataDbUtil.getNumberOfSpecificDateSchedules(),
        is(3));
    assertThat(
        "It should have 4 recurring schedules.",
        testDataDbUtil.getNumberOfRecurringSchedules(),
        is(4));

    Mockito.verify(scheduler, Mockito.never()).deleteJob(any());
  }

  private void assertNoSchedulesFound(ResultActions resultActions) throws Exception {
    resultActions.andExpect(content().string(is(emptyString())));
    resultActions.andExpect(header().doesNotExist("Content-type"));
    resultActions.andExpect(status().isNotFound());
  }

  private void assertResponseForCreateSchedules(
      ResultActions resultActions, ResultMatcher expectedStatus) throws Exception {
    resultActions.andExpect(content().string(is(emptyString())));
    resultActions.andExpect(header().doesNotExist("Content-type"));
    resultActions.andExpect(expectedStatus);
  }

  private void assertSchedulesFoundEquals(
      ApplicationSchedules applicationSchedules,
      String appId,
      ResultActions resultActions,
      int expectedSpecificDateSchedulesTobeFound,
      int expectedRecurringSchedulesTobeFound)
      throws Exception {

    resultActions.andExpect(status().isOk());
    resultActions.andExpect(content().contentTypeCompatibleWith(MediaType.APPLICATION_JSON));

    assertSpecificDateScheduleFoundEquals(
        expectedSpecificDateSchedulesTobeFound,
        appId,
        applicationSchedules.getSchedules().getSpecificDate());
    assertRecurringDateScheduleFoundEquals(
        expectedRecurringSchedulesTobeFound,
        appId,
        applicationSchedules.getSchedules().getRecurringSchedule());
  }

  private void assertSpecificDateScheduleFoundEquals(
      int expectedSchedulesTobeFound,
      String expectedAppId,
      List<SpecificDateScheduleEntity> specificDateScheduls) {
    if (specificDateScheduls == null) {
      assertEquals(expectedSchedulesTobeFound, 0);
    } else {
      assertEquals(expectedSchedulesTobeFound, specificDateScheduls.size());
      for (ScheduleEntity entity : specificDateScheduls) {
        assertEquals(expectedAppId, entity.getAppId());
      }
    }
  }

  private void assertRecurringDateScheduleFoundEquals(
      int expectedRecurringSchedulesTobeFound,
      String expectedAppId,
      List<RecurringScheduleEntity> recurringSchedule) {
    if (recurringSchedule == null) {
      assertEquals(expectedRecurringSchedulesTobeFound, 0);
    } else {
      assertEquals(expectedRecurringSchedulesTobeFound, recurringSchedule.size());
      for (ScheduleEntity entity : recurringSchedule) {
        assertEquals(expectedAppId, entity.getAppId());
      }
    }
  }

  private void assertErrorMessages(ResultActions resultActions, String... expectedErrorMessages)
      throws Exception {
    resultActions.andExpect(
        jsonPath("$").value(Matchers.containsInAnyOrder(expectedErrorMessages)));
    resultActions.andExpect(jsonPath("$").isArray());
    resultActions.andExpect(content().contentTypeCompatibleWith(MediaType.APPLICATION_JSON));
    resultActions.andExpect(status().isBadRequest());
  }

  private void assertSchedulesAreDeleted(ResultActions resultActions) throws Exception {
    resultActions.andExpect(content().string(is(emptyString())));
    resultActions.andExpect(header().doesNotExist("Content-type"));
    resultActions.andExpect(status().isNoContent());
  }

  private ApplicationSchedules getApplicationSchedulesFromResultActions(ResultActions resultActions)
      throws IOException {
    ObjectMapper mapper = new ObjectMapper();
    mapper.setDateFormat(DateFormat.getDateInstance(DateFormat.LONG));
    return mapper.readValue(
        resultActions.andReturn().getResponse().getContentAsString(), ApplicationSchedules.class);
  }
}
