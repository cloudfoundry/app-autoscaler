package org.cloudfoundry.autoscaler.scheduler.service;

import static org.hamcrest.Matchers.containsString;
import static org.hamcrest.Matchers.is;
import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertThat;
import static org.junit.Assert.assertTrue;
import static org.junit.Assert.fail;
import static org.mockito.Matchers.any;
import static org.mockito.Matchers.anyLong;
import static org.mockito.Matchers.anyObject;
import static org.mockito.Matchers.anyString;
import static org.mockito.Matchers.eq;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.times;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.client.match.MockRestRequestMatchers.method;
import static org.springframework.test.web.client.match.MockRestRequestMatchers.requestTo;
import static org.springframework.test.web.client.response.MockRestResponseCreators.withNoContent;
import static org.springframework.test.web.client.response.MockRestResponseCreators.withStatus;

import com.fasterxml.jackson.core.JsonProcessingException;
import java.util.ArrayList;
import java.util.List;
import org.apache.logging.log4j.Level;
import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.core.Appender;
import org.apache.logging.log4j.core.LogEvent;
import org.apache.logging.log4j.core.LoggerContext;
import org.apache.logging.log4j.core.config.Configuration;
import org.apache.logging.log4j.core.config.LoggerConfig;
import org.cloudfoundry.autoscaler.scheduler.dao.ActiveScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.dao.PolicyJsonDao;
import org.cloudfoundry.autoscaler.scheduler.dao.RecurringScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.dao.SpecificDateScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.ActiveScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.PolicyJsonEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationSchedules;
import org.cloudfoundry.autoscaler.scheduler.rest.model.Schedules;
import org.cloudfoundry.autoscaler.scheduler.rest.model.SynchronizeResult;
import org.cloudfoundry.autoscaler.scheduler.util.ApplicationPolicyBuilder;
import org.cloudfoundry.autoscaler.scheduler.util.PolicyJsonEntityBuilder;
import org.cloudfoundry.autoscaler.scheduler.util.RecurringScheduleEntitiesBuilder;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleBuilder;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleTypeEnum;
import org.cloudfoundry.autoscaler.scheduler.util.SpecificDateScheduleEntitiesBuilder;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataDbUtil;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.DatabaseValidationException;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.SchedulerInternalException;
import org.cloudfoundry.autoscaler.scheduler.util.error.ValidationErrorResult;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.ArgumentCaptor;
import org.mockito.Captor;
import org.mockito.Mock;
import org.mockito.Mockito;
import org.quartz.SchedulerException;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.boot.test.mock.mockito.SpyBean;
import org.springframework.http.HttpMethod;
import org.springframework.http.HttpStatus;
import org.springframework.test.context.junit4.SpringRunner;
import org.springframework.test.web.client.ExpectedCount;
import org.springframework.test.web.client.MockRestServiceServer;
import org.springframework.web.client.ResourceAccessException;
import org.springframework.web.client.RestOperations;
import org.springframework.web.client.RestTemplate;

@RunWith(SpringRunner.class)
@SpringBootTest
public class ScheduleManagerTest {

  @Autowired private ScheduleManager scheduleManager;

  @MockBean private PolicyJsonDao policyJsonDao;

  @MockBean private SpecificDateScheduleDao specificDateScheduleDao;

  @MockBean private RecurringScheduleDao recurringScheduleDao;

  @MockBean private ActiveScheduleDao activeScheduleDao;

  @MockBean private ScheduleJobManager scheduleJobManager;

  @Autowired private MessageBundleResourceHelper messageBundleResourceHelper;

  @Autowired private ValidationErrorResult validationErrorResult;

  @Autowired private TestDataDbUtil testDataDbUtil;

  @SpyBean private RestOperations restOperations;

  @Value("${autoscaler.scalingengine.url}")
  private String scalingEngineUrl;

  private MockRestServiceServer mockServer;

  @Mock private Appender mockAppender;

  @Captor private ArgumentCaptor<LogEvent> logCaptor;

  @Before
  public void before() throws SchedulerException {
    testDataDbUtil.cleanupData();

    Mockito.reset(policyJsonDao);
    Mockito.reset(specificDateScheduleDao);
    Mockito.reset(recurringScheduleDao);
    Mockito.reset(activeScheduleDao);
    Mockito.reset(mockAppender);
    Mockito.reset(restOperations);
    mockServer = MockRestServiceServer.createServer((RestTemplate) restOperations);

    Mockito.when(mockAppender.getName()).thenReturn("MockAppender");
    Mockito.when(mockAppender.isStarted()).thenReturn(true);
    Mockito.when(mockAppender.isStopped()).thenReturn(false);

    setLogLevel(Level.INFO);
  }

  @Test
  public void testGetAllSchedules_with_no_schedules() {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    Mockito.when(specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(eq(appId)))
        .thenReturn(new ArrayList<>());
    Mockito.when(recurringScheduleDao.findAllRecurringSchedulesByAppId(eq(appId)))
        .thenReturn(new ArrayList<>());

    Schedules scalingSchedules = scheduleManager.getAllSchedules(appId).getSchedules();

    assertFalse(scalingSchedules.hasSchedules());
    Mockito.verify(specificDateScheduleDao, Mockito.times(1))
        .findAllSpecificDateSchedulesByAppId(appId);
    Mockito.verify(recurringScheduleDao, Mockito.times(1)).findAllRecurringSchedulesByAppId(appId);
  }

  @Test
  public void testGetAllSchedules() {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    List<SpecificDateScheduleEntity> specificDateScheduleEntities =
        new SpecificDateScheduleEntitiesBuilder(3).setAppid(appId).setScheduleId().build();
    List<RecurringScheduleEntity> recurringScheduleEntities =
        new RecurringScheduleEntitiesBuilder(2, 2).setAppId(appId).setScheduleId().build();

    Mockito.when(specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(eq(appId)))
        .thenReturn(specificDateScheduleEntities);
    Mockito.when(recurringScheduleDao.findAllRecurringSchedulesByAppId(eq(appId)))
        .thenReturn(recurringScheduleEntities);

    Schedules scalingSchedules = scheduleManager.getAllSchedules(appId).getSchedules();

    Mockito.verify(specificDateScheduleDao, Mockito.times(1))
        .findAllSpecificDateSchedulesByAppId(appId);
    Mockito.verify(recurringScheduleDao, Mockito.times(1)).findAllRecurringSchedulesByAppId(appId);
    assertThat(
        "Both specific schedules are equal",
        scalingSchedules.getSpecificDate(),
        is(specificDateScheduleEntities));
    assertThat(
        "Both recurring schedules are equal",
        scalingSchedules.getRecurringSchedule(),
        is(recurringScheduleEntities));
  }

  @Test
  public void testCreateSchedules_with_specificDateSchedule() {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String guid = TestDataSetupHelper.generateGuid();
    int noOfSpecificDateSchedules = 1;

    Schedules schedules =
        TestDataSetupHelper.generateSchedulesWithEntitiesOnly(
            appId, guid, false, noOfSpecificDateSchedules, 0, 0);

    SpecificDateScheduleEntity specificDateScheduleEntity =
        new SpecificDateScheduleEntitiesBuilder(1).setAppid(appId).setScheduleId().build().get(0);
    Mockito.when(specificDateScheduleDao.create(Mockito.anyObject()))
        .thenReturn(specificDateScheduleEntity);

    scheduleManager.createSchedules(schedules);

    assertCreateSchedules(
        schedules, specificDateScheduleEntity, null, noOfSpecificDateSchedules, 0, 0);
  }

  @Test
  public void testCreateSchedules_with_dayOfMonth_recurringSchedule() {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String guid = TestDataSetupHelper.generateGuid();
    int noOfDOMRecurringSchedules = 1;
    int noOfDOWRecurringSchedules = 0;

    Schedules schedules =
        TestDataSetupHelper.generateSchedulesWithEntitiesOnly(
            appId, guid, false, 0, noOfDOMRecurringSchedules, noOfDOWRecurringSchedules);

    RecurringScheduleEntity recurringScheduleEntity =
        new RecurringScheduleEntitiesBuilder(1, 0).setAppId(appId).setScheduleId().build().get(0);
    Mockito.when(recurringScheduleDao.create(Mockito.anyObject()))
        .thenReturn(recurringScheduleEntity);

    scheduleManager.createSchedules(schedules);

    assertCreateSchedules(
        schedules,
        null,
        recurringScheduleEntity,
        0,
        noOfDOMRecurringSchedules,
        noOfDOWRecurringSchedules);
  }

  @Test
  public void testCreateSchedules_with_dayOfWeek_recurringSchedule() {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String guid = TestDataSetupHelper.generateGuid();
    int noOfDOMRecurringSchedules = 0;
    int noOfDOWRecurringSchedules = 1;

    Schedules schedules =
        TestDataSetupHelper.generateSchedulesWithEntitiesOnly(
            appId, guid, false, 0, noOfDOMRecurringSchedules, noOfDOWRecurringSchedules);

    RecurringScheduleEntity recurringScheduleEntity =
        new RecurringScheduleEntitiesBuilder(1, 0).setAppId(appId).setScheduleId().build().get(0);
    Mockito.when(recurringScheduleDao.create(Mockito.anyObject()))
        .thenReturn(recurringScheduleEntity);

    scheduleManager.createSchedules(schedules);

    assertCreateSchedules(
        schedules,
        null,
        recurringScheduleEntity,
        0,
        noOfDOMRecurringSchedules,
        noOfDOWRecurringSchedules);
  }

  @Test
  public void testCreateSchedules_with_dayOfWeek_recurringSchedule_compensatoryRequired() {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String guid = TestDataSetupHelper.generateGuid();
    int noOfDOMRecurringSchedules = 0;
    int noOfDOWRecurringSchedules = 1;

    Schedules schedules =
        TestDataSetupHelper.generateSchedulesWithEntitiesOnly(
            appId, guid, false, 0, noOfDOMRecurringSchedules, noOfDOWRecurringSchedules);

    RecurringScheduleEntity recurringSchedule = schedules.getRecurringSchedule().get(0);
    recurringSchedule.setStartTime(TestDataSetupHelper.getZoneTimeWithOffset(-5));
    recurringSchedule.setEndTime(TestDataSetupHelper.getZoneTimeWithOffset(5));

    RecurringScheduleEntity recurringScheduleEntity =
        new RecurringScheduleEntitiesBuilder(1, 0).setAppId(appId).setScheduleId().build().get(0);
    Mockito.when(recurringScheduleDao.create(Mockito.anyObject()))
        .thenReturn(recurringScheduleEntity);

    SpecificDateScheduleEntity specificDateScheduleEntity =
        new SpecificDateScheduleEntitiesBuilder(1).setAppid(appId).setScheduleId().build().get(0);
    Mockito.when(specificDateScheduleDao.create(Mockito.anyObject()))
        .thenReturn(specificDateScheduleEntity);

    scheduleManager.createSchedules(schedules);

    assertCreateSchedules(
        schedules,
        specificDateScheduleEntity,
        recurringScheduleEntity,
        1,
        noOfDOMRecurringSchedules,
        noOfDOWRecurringSchedules);
  }

  @Test
  public void testCreateSchedules_with_dayOfMonth_recurringSchedule_compensatoryRequired() {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String guid = TestDataSetupHelper.generateGuid();
    int noOfDOMRecurringSchedules = 1;
    int noOfDOWRecurringSchedules = 0;

    Schedules schedules =
        TestDataSetupHelper.generateSchedulesWithEntitiesOnly(
            appId, guid, false, 0, noOfDOMRecurringSchedules, noOfDOWRecurringSchedules);

    RecurringScheduleEntity recurringSchedule = schedules.getRecurringSchedule().get(0);
    recurringSchedule.setStartTime(TestDataSetupHelper.getZoneTimeWithOffset(-5));
    recurringSchedule.setEndTime(TestDataSetupHelper.getZoneTimeWithOffset(5));

    RecurringScheduleEntity recurringScheduleEntity =
        new RecurringScheduleEntitiesBuilder(1, 0).setAppId(appId).setScheduleId().build().get(0);
    Mockito.when(recurringScheduleDao.create(Mockito.anyObject()))
        .thenReturn(recurringScheduleEntity);

    SpecificDateScheduleEntity specificDateScheduleEntity =
        new SpecificDateScheduleEntitiesBuilder(1).setAppid(appId).setScheduleId().build().get(0);
    Mockito.when(specificDateScheduleDao.create(Mockito.anyObject()))
        .thenReturn(specificDateScheduleEntity);

    scheduleManager.createSchedules(schedules);

    assertCreateSchedules(
        schedules,
        specificDateScheduleEntity,
        recurringScheduleEntity,
        1,
        noOfDOMRecurringSchedules,
        noOfDOWRecurringSchedules);
  }

  @Test
  public void
      testCreateSchedules_with_dayOfWeek_recurringSchedule_compensatoryNotRequiredPerStartDate() {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String guid = TestDataSetupHelper.generateGuid();
    int noOfDOMRecurringSchedules = 1;
    int noOfDOWRecurringSchedules = 0;

    Schedules schedules =
        TestDataSetupHelper.generateSchedulesWithEntitiesOnly(
            appId, guid, false, 0, noOfDOMRecurringSchedules, noOfDOWRecurringSchedules);

    RecurringScheduleEntity recurringSchedule = schedules.getRecurringSchedule().get(0);
    recurringSchedule.setStartTime(TestDataSetupHelper.getZoneTimeWithOffset(-5));
    recurringSchedule.setEndTime(TestDataSetupHelper.getZoneTimeWithOffset(5));
    recurringSchedule.setStartDate(TestDataSetupHelper.getZoneDateNow().plusDays(2));

    RecurringScheduleEntity recurringScheduleEntity =
        new RecurringScheduleEntitiesBuilder(1, 0).setAppId(appId).setScheduleId().build().get(0);
    Mockito.when(recurringScheduleDao.create(Mockito.anyObject()))
        .thenReturn(recurringScheduleEntity);

    scheduleManager.createSchedules(schedules);

    assertCreateSchedules(
        schedules,
        null,
        recurringScheduleEntity,
        0,
        noOfDOMRecurringSchedules,
        noOfDOWRecurringSchedules);
  }

  @Test
  public void testCreateSchedules() {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String guid = TestDataSetupHelper.generateGuid();
    int noOfSpecificDateSchedules = 3;
    int noOfDOMRecurringSchedules = 3;
    int noOfDOWRecurringSchedules = 3;

    Schedules schedules =
        TestDataSetupHelper.generateSchedulesWithEntitiesOnly(
            appId,
            guid,
            false,
            noOfSpecificDateSchedules,
            noOfDOMRecurringSchedules,
            noOfDOWRecurringSchedules);

    SpecificDateScheduleEntity specificDateScheduleEntity =
        new SpecificDateScheduleEntitiesBuilder(1).setAppid(appId).setScheduleId().build().get(0);
    RecurringScheduleEntity recurringScheduleEntity =
        new RecurringScheduleEntitiesBuilder(1, 0).setAppId(appId).setScheduleId().build().get(0);
    Mockito.when(specificDateScheduleDao.create(Mockito.anyObject()))
        .thenReturn(specificDateScheduleEntity);
    Mockito.when(recurringScheduleDao.create(Mockito.anyObject()))
        .thenReturn(recurringScheduleEntity);

    scheduleManager.createSchedules(schedules);

    assertCreateSchedules(
        schedules,
        specificDateScheduleEntity,
        recurringScheduleEntity,
        noOfSpecificDateSchedules,
        noOfDOMRecurringSchedules,
        noOfDOWRecurringSchedules);
  }

  @Test
  public void testCreateSpecificDateSchedule_throw_DatabaseValidationException() {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String guid = TestDataSetupHelper.generateGuid();
    Schedules schedules =
        TestDataSetupHelper.generateSchedulesWithEntitiesOnly(appId, guid, false, 1, 0, 0);

    Mockito.when(specificDateScheduleDao.create(Mockito.anyObject()))
        .thenThrow(new DatabaseValidationException("test exception"));

    try {
      scheduleManager.createSchedules(schedules);
      fail("Should fail");
    } catch (SchedulerInternalException e) {
      String message =
          messageBundleResourceHelper.lookupMessage(
              "database.error.create.failed", "app_id=" + appId);

      for (String errorMessage : validationErrorResult.getAllErrorMessages()) {
        assertEquals(message, errorMessage);
      }
    }

    Mockito.verify(scheduleJobManager, Mockito.never()).createSimpleJob(Mockito.anyObject());
  }

  @Test
  public void testCreateRecurringSchedule_throw_DatabaseValidationException() {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String guid = TestDataSetupHelper.generateGuid();
    Schedules schedules =
        TestDataSetupHelper.generateSchedulesWithEntitiesOnly(appId, guid, false, 0, 1, 0);

    Mockito.when(recurringScheduleDao.create(Mockito.anyObject()))
        .thenThrow(new DatabaseValidationException("test exception"));

    try {
      scheduleManager.createSchedules(schedules);
      fail("Should fail");
    } catch (SchedulerInternalException e) {
      String message =
          messageBundleResourceHelper.lookupMessage(
              "database.error.create.failed", "app_id=" + appId);

      for (String errorMessage : validationErrorResult.getAllErrorMessages()) {
        assertEquals(message, errorMessage);
      }
    }

    Mockito.verify(scheduleJobManager, Mockito.never()).createCronJob(Mockito.anyObject());
  }

  @Test
  public void testFindAllSpecificDateSchedule_throw_DatabaseValidationException() {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    Mockito.when(specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(Mockito.anyString()))
        .thenThrow(new DatabaseValidationException("test exception"));

    try {
      scheduleManager.getAllSchedules("appId1");
    } catch (SchedulerInternalException sie) {
      String message =
          messageBundleResourceHelper.lookupMessage("database.error.get.failed", "app_id=" + appId);

      for (String errorMessage : validationErrorResult.getAllErrorMessages()) {
        assertEquals(message, errorMessage);
      }
    }
  }

  @Test
  public void testFindAllRecurringSchedule_throw_DatabaseValidationException() {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    Mockito.when(recurringScheduleDao.findAllRecurringSchedulesByAppId(Mockito.anyString()))
        .thenThrow(new DatabaseValidationException("test exception"));

    try {
      scheduleManager.getAllSchedules("appId1");
    } catch (SchedulerInternalException sie) {
      String message =
          messageBundleResourceHelper.lookupMessage("database.error.get.failed", "app_id=" + appId);

      for (String errorMessage : validationErrorResult.getAllErrorMessages()) {
        assertEquals(message, errorMessage);
      }
    }
  }

  @Test
  public void testDeleteSchedules() {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    long scheduleId = 1L;

    ActiveScheduleEntity activeScheduleEntity = new ActiveScheduleEntity();
    activeScheduleEntity.setAppId(appId);
    activeScheduleEntity.setId(scheduleId);
    List<ActiveScheduleEntity> activeScheduleEntities = new ArrayList<>();
    activeScheduleEntities.add(activeScheduleEntity);

    List<SpecificDateScheduleEntity> specificDateScheduleEntities =
        new SpecificDateScheduleEntitiesBuilder(2).setAppid(appId).setScheduleId().build();
    List<RecurringScheduleEntity> recurringScheduleEntities =
        new RecurringScheduleEntitiesBuilder(1, 1).setAppId(appId).setScheduleId().build();

    Mockito.when(specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(appId))
        .thenReturn(specificDateScheduleEntities);
    Mockito.when(recurringScheduleDao.findAllRecurringSchedulesByAppId(appId))
        .thenReturn(recurringScheduleEntities);

    Mockito.when(activeScheduleDao.findByAppId(appId)).thenReturn(activeScheduleEntities);

    String scalingEnginePathActiveSchedule =
        scalingEngineUrl + "/v1/apps/" + appId + "/active_schedules/" + scheduleId;
    mockServer
        .expect(ExpectedCount.times(1), requestTo(scalingEnginePathActiveSchedule))
        .andExpect(method(HttpMethod.DELETE))
        .andRespond(withNoContent());

    scheduleManager.deleteSchedules(appId);

    for (SpecificDateScheduleEntity specificDateScheduleEntity : specificDateScheduleEntities) {
      Mockito.verify(specificDateScheduleDao, Mockito.times(1)).delete(specificDateScheduleEntity);
      Mockito.verify(scheduleJobManager, Mockito.times(1))
          .deleteJob(
              specificDateScheduleEntity.getAppId(),
              specificDateScheduleEntity.getId(),
              ScheduleTypeEnum.SPECIFIC_DATE);
    }
    for (RecurringScheduleEntity recurringScheduleEntity : recurringScheduleEntities) {
      Mockito.verify(recurringScheduleDao, Mockito.times(1)).delete(recurringScheduleEntity);
      Mockito.verify(scheduleJobManager, Mockito.times(1))
          .deleteJob(
              recurringScheduleEntity.getAppId(),
              recurringScheduleEntity.getId(),
              ScheduleTypeEnum.RECURRING);
    }
    Mockito.verify(activeScheduleDao, Mockito.times(1)).deleteActiveSchedulesByAppId(appId);

    mockServer.verify();
  }

  @Test
  public void testNotifyScalingEngine_when_ResourceAccessException() throws Exception {
    setLogLevel(Level.ERROR);

    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    long scheduleId = 1L;

    ActiveScheduleEntity activeScheduleEntity = new ActiveScheduleEntity();
    activeScheduleEntity.setAppId(appId);
    activeScheduleEntity.setId(scheduleId);
    List<ActiveScheduleEntity> activeScheduleEntities = new ArrayList<>();
    activeScheduleEntities.add(activeScheduleEntity);

    List<SpecificDateScheduleEntity> specificDateScheduleEntities =
        new SpecificDateScheduleEntitiesBuilder(2).setAppid(appId).setScheduleId().build();

    Mockito.when(specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(appId))
        .thenReturn(specificDateScheduleEntities);
    Mockito.doNothing()
        .when(scheduleJobManager)
        .deleteJob(Mockito.anyString(), Mockito.anyLong(), Mockito.anyObject());

    Mockito.when(activeScheduleDao.findByAppId(appId)).thenReturn(activeScheduleEntities);
    Mockito.doThrow(new ResourceAccessException("test resource access exception"))
        .when(restOperations)
        .delete(Mockito.anyString());

    try {
      scheduleManager.deleteSchedules(appId);
      fail("Should fail");
    } catch (SchedulerInternalException sie) {
      String expectedMessage =
          messageBundleResourceHelper.lookupMessage(
              "scalingengine.notification.error",
              "test resource access exception",
              appId,
              scheduleId,
              "delete");
      assertThat(sie.getMessage(), is(expectedMessage));
      assertTrue(sie.getCause().getClass().equals(ResourceAccessException.class));
    }
  }

  @Test
  public void testDeleteSchedules_when_activeSchedule_not_found_in_scalingEngine() {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    long scheduleId = 1L;

    ActiveScheduleEntity activeScheduleEntity = new ActiveScheduleEntity();
    activeScheduleEntity.setAppId(appId);
    activeScheduleEntity.setId(scheduleId);
    List<ActiveScheduleEntity> activeScheduleEntities = new ArrayList<>();
    activeScheduleEntities.add(activeScheduleEntity);

    List<SpecificDateScheduleEntity> specificDateScheduleEntities =
        new SpecificDateScheduleEntitiesBuilder(2).setAppid(appId).setScheduleId().build();
    List<RecurringScheduleEntity> recurringScheduleEntities =
        new RecurringScheduleEntitiesBuilder(1, 1).setAppId(appId).setScheduleId().build();

    Mockito.when(specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(appId))
        .thenReturn(specificDateScheduleEntities);
    Mockito.when(recurringScheduleDao.findAllRecurringSchedulesByAppId(appId))
        .thenReturn(recurringScheduleEntities);

    Mockito.when(activeScheduleDao.findByAppId(appId)).thenReturn(activeScheduleEntities);

    String scalingEnginePathActiveSchedule =
        scalingEngineUrl + "/v1/apps/" + appId + "/active_schedules/" + scheduleId;
    mockServer
        .expect(ExpectedCount.times(1), requestTo(scalingEnginePathActiveSchedule))
        .andExpect(method(HttpMethod.DELETE))
        .andRespond(withStatus(HttpStatus.OK));

    scheduleManager.deleteSchedules(appId);

    Mockito.verify(mockAppender, Mockito.atLeastOnce()).append(logCaptor.capture());

    for (SpecificDateScheduleEntity specificDateScheduleEntity : specificDateScheduleEntities) {
      Mockito.verify(specificDateScheduleDao, Mockito.times(1)).delete(specificDateScheduleEntity);
      Mockito.verify(scheduleJobManager, Mockito.times(1))
          .deleteJob(
              specificDateScheduleEntity.getAppId(),
              specificDateScheduleEntity.getId(),
              ScheduleTypeEnum.SPECIFIC_DATE);
    }
    for (RecurringScheduleEntity recurringScheduleEntity : recurringScheduleEntities) {
      Mockito.verify(recurringScheduleDao, Mockito.times(1)).delete(recurringScheduleEntity);
      Mockito.verify(scheduleJobManager, Mockito.times(1))
          .deleteJob(
              recurringScheduleEntity.getAppId(),
              recurringScheduleEntity.getId(),
              ScheduleTypeEnum.RECURRING);
    }
    Mockito.verify(activeScheduleDao, Mockito.times(1)).deleteActiveSchedulesByAppId(appId);

    mockServer.verify();
  }

  @Test
  public void testDeleteSchedules_when_internal_server_error_in_scalingEngine() {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    long scheduleId = 1L;

    ActiveScheduleEntity activeScheduleEntity = new ActiveScheduleEntity();
    activeScheduleEntity.setAppId(appId);
    activeScheduleEntity.setId(scheduleId);
    List<ActiveScheduleEntity> activeScheduleEntities = new ArrayList<>();
    activeScheduleEntities.add(activeScheduleEntity);

    List<SpecificDateScheduleEntity> specificDateScheduleEntities =
        new SpecificDateScheduleEntitiesBuilder(2).setAppid(appId).setScheduleId().build();
    List<RecurringScheduleEntity> recurringScheduleEntities =
        new RecurringScheduleEntitiesBuilder(1, 1).setAppId(appId).setScheduleId().build();

    Mockito.when(specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(appId))
        .thenReturn(specificDateScheduleEntities);
    Mockito.when(recurringScheduleDao.findAllRecurringSchedulesByAppId(appId))
        .thenReturn(recurringScheduleEntities);

    Mockito.when(activeScheduleDao.findByAppId(appId)).thenReturn(activeScheduleEntities);

    String scalingEnginePathActiveSchedule =
        scalingEngineUrl + "/v1/apps/" + appId + "/active_schedules/" + scheduleId;
    mockServer
        .expect(ExpectedCount.times(1), requestTo(scalingEnginePathActiveSchedule))
        .andExpect(method(HttpMethod.DELETE))
        .andRespond(
            withStatus(HttpStatus.INTERNAL_SERVER_ERROR).body("test internal server error"));

    try {
      scheduleManager.deleteSchedules(appId);
      fail("Should fail");
    } catch (SchedulerInternalException sie) {
      String expectedMessage =
          messageBundleResourceHelper.lookupMessage(
              "scalingengine.notification.error",
              "test internal server error",
              appId,
              scheduleId,
              "delete");
      assertThat(sie.getMessage(), containsString(expectedMessage));
    }
    mockServer.verify();
  }

  @Test
  public void testDeleteSchedules_without_any_schedules() {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];

    List<SpecificDateScheduleEntity> specificDateScheduleEntities = new ArrayList<>();

    List<RecurringScheduleEntity> recurringScheduleEntities = new ArrayList<>();

    Mockito.when(specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(appId))
        .thenReturn(specificDateScheduleEntities);
    Mockito.when(recurringScheduleDao.findAllRecurringSchedulesByAppId(appId))
        .thenReturn(recurringScheduleEntities);

    scheduleManager.deleteSchedules(appId);

    Mockito.verify(specificDateScheduleDao, Mockito.never()).delete(Mockito.anyObject());
    Mockito.verify(scheduleJobManager, Mockito.never())
        .deleteJob(Mockito.anyString(), Mockito.anyLong(), Mockito.anyObject());

    Mockito.verify(recurringScheduleDao, Mockito.never()).delete(Mockito.anyObject());
    Mockito.verify(scheduleJobManager, Mockito.never())
        .deleteJob(Mockito.anyString(), Mockito.anyLong(), Mockito.anyObject());

    Mockito.verify(activeScheduleDao, Mockito.times(1)).deleteActiveSchedulesByAppId(appId);
  }

  @Test
  public void testDeleteSpecificDateSchedules_throw_DatabaseValidationException() {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];

    List<SpecificDateScheduleEntity> specificDateScheduleEntities =
        new SpecificDateScheduleEntitiesBuilder(2).setAppid(appId).setScheduleId().build();
    List<RecurringScheduleEntity> recurringScheduleEntities =
        new RecurringScheduleEntitiesBuilder(1, 1).setAppId(appId).setScheduleId().build();

    Mockito.when(specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(appId))
        .thenReturn(specificDateScheduleEntities);
    Mockito.when(recurringScheduleDao.findAllRecurringSchedulesByAppId(appId))
        .thenReturn(recurringScheduleEntities);

    Mockito.doThrow(new DatabaseValidationException("test exception"))
        .when(specificDateScheduleDao)
        .delete(Mockito.anyObject());

    try {
      scheduleManager.deleteSchedules(appId);
      fail("Should fail");
    } catch (SchedulerInternalException sie) {
      String message =
          messageBundleResourceHelper.lookupMessage(
              "database.error.delete.failed", "app_id=" + appId);

      for (String errorMessage : validationErrorResult.getAllErrorMessages()) {
        assertEquals(message, errorMessage);
      }
    }

    Mockito.verify(scheduleJobManager, Mockito.never())
        .deleteJob(Mockito.anyString(), Mockito.anyLong(), eq(ScheduleTypeEnum.SPECIFIC_DATE));
    Mockito.verify(activeScheduleDao, Mockito.never())
        .deleteActiveSchedulesByAppId(Mockito.anyString());
  }

  @Test
  public void testDeleteRecurringSchedules_throw_DatabaseValidationException() {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];

    List<SpecificDateScheduleEntity> specificDateScheduleEntities =
        new SpecificDateScheduleEntitiesBuilder(2).setAppid(appId).setScheduleId().build();
    List<RecurringScheduleEntity> recurringScheduleEntities =
        new RecurringScheduleEntitiesBuilder(1, 1).setAppId(appId).setScheduleId().build();

    Mockito.when(specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(appId))
        .thenReturn(specificDateScheduleEntities);
    Mockito.when(recurringScheduleDao.findAllRecurringSchedulesByAppId(appId))
        .thenReturn(recurringScheduleEntities);

    Mockito.doThrow(new DatabaseValidationException("test exception"))
        .when(recurringScheduleDao)
        .delete(Mockito.anyObject());

    try {
      scheduleManager.deleteSchedules(appId);
      fail("Should fail");
    } catch (SchedulerInternalException sie) {
      String message =
          messageBundleResourceHelper.lookupMessage(
              "database.error.delete.failed", "app_id=" + appId);

      for (String errorMessage : validationErrorResult.getAllErrorMessages()) {
        assertEquals(message, errorMessage);
      }
    }

    Mockito.verify(scheduleJobManager, Mockito.never())
        .deleteJob(Mockito.anyString(), Mockito.anyLong(), eq(ScheduleTypeEnum.RECURRING));
    Mockito.verify(activeScheduleDao, Mockito.never())
        .deleteActiveSchedulesByAppId(Mockito.anyString());
  }

  @Test
  public void testDeleteActiveSchedules_throw_DatabaseValidationException() {

    String appId = TestDataSetupHelper.generateAppIds(1)[0];

    List<SpecificDateScheduleEntity> specificDateScheduleEntities =
        new SpecificDateScheduleEntitiesBuilder(2).setAppid(appId).setScheduleId().build();
    List<RecurringScheduleEntity> recurringScheduleEntities =
        new RecurringScheduleEntitiesBuilder(1, 1).setAppId(appId).setScheduleId().build();

    Mockito.when(specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(appId))
        .thenReturn(specificDateScheduleEntities);
    Mockito.when(recurringScheduleDao.findAllRecurringSchedulesByAppId(appId))
        .thenReturn(recurringScheduleEntities);
    // Mock the exception when deleting active schedule
    Mockito.doThrow(new DatabaseValidationException("test exception"))
        .when(activeScheduleDao)
        .deleteActiveSchedulesByAppId(appId);

    try {
      scheduleManager.deleteSchedules(appId);
      fail("Should fail");
    } catch (SchedulerInternalException e) {
      String message =
          messageBundleResourceHelper.lookupMessage(
              "database.error.delete.failed", "app_id=" + appId);

      for (String errorMessage : validationErrorResult.getAllErrorMessages()) {
        assertEquals(message, errorMessage);
      }
    }
    mockServer.verify();
  }

  @Test
  public void testSynchronizeSchedules_with_no_policy_and_no_schedules() {

    when(policyJsonDao.getAllPolicies()).thenReturn(new ArrayList<PolicyJsonEntity>());
    when(recurringScheduleDao.getDistinctAppIdAndGuidList()).thenReturn(new ArrayList<Object[]>());
    when(specificDateScheduleDao.getDistinctAppIdAndGuidList())
        .thenReturn(new ArrayList<Object[]>());
    SynchronizeResult result = scheduleManager.synchronizeSchedules();

    assertThat("It should do nothing", result, is(new SynchronizeResult(0, 0, 0)));
    verify(policyJsonDao, times(1)).getAllPolicies();
    verify(recurringScheduleDao, times(1)).getDistinctAppIdAndGuidList();
    verify(specificDateScheduleDao, times(1)).getDistinctAppIdAndGuidList();

    verify(recurringScheduleDao, never()).create(any());
    verify(specificDateScheduleDao, never()).create(any());
    verify(activeScheduleDao, never()).create(any());

    verify(recurringScheduleDao, never()).delete(any());
    verify(specificDateScheduleDao, never()).delete(any());
    verify(activeScheduleDao, never()).deleteActiveSchedulesByAppId(anyString());
    verify(scheduleJobManager, never()).createCronJob(any());
    verify(scheduleJobManager, never()).createSimpleJob(any());
    verify(scheduleJobManager, never()).deleteJob(anyString(), anyLong(), any());
  }

  @Test
  public void testSynchronizeSchedules_with_existed_policy_and_no_schedule()
      throws JsonProcessingException {

    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String guid = TestDataSetupHelper.generateGuid();
    int noOfSpecificDateSchedules = 3;
    int noOfDOMRecurringSchedules = 3;
    int noOfDOWRecurringSchedules = 3;

    List<RecurringScheduleEntity> recurringEntities =
        new RecurringScheduleEntitiesBuilder(noOfDOMRecurringSchedules, noOfDOWRecurringSchedules)
            .setAppId(appId)
            .setGuid(guid)
            .setTimeZone(TestDataSetupHelper.timeZone)
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .setScheduleId()
            .build();
    List<SpecificDateScheduleEntity> specificDateEntities =
        new SpecificDateScheduleEntitiesBuilder(noOfSpecificDateSchedules)
            .setAppid(appId)
            .setGuid(guid)
            .setTimeZone(TestDataSetupHelper.timeZone)
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .setScheduleId()
            .build();
    Schedules schedules =
        new ScheduleBuilder()
            .setSpecificDate(specificDateEntities)
            .setRecurringSchedule(recurringEntities)
            .setTimeZone(TestDataSetupHelper.timeZone)
            .build();
    ApplicationSchedules applicationSchedule =
        new ApplicationPolicyBuilder(1, 5).setSchedules(schedules).build();

    List<PolicyJsonEntity> policyJsonList =
        new ArrayList<PolicyJsonEntity>() {
          {
            add(new PolicyJsonEntityBuilder(appId, guid, applicationSchedule).build());
          }
        };
    when(policyJsonDao.getAllPolicies()).thenReturn(policyJsonList);
    when(specificDateScheduleDao.getDistinctAppIdAndGuidList())
        .thenReturn(new ArrayList<Object[]>());
    when(recurringScheduleDao.getDistinctAppIdAndGuidList()).thenReturn(new ArrayList<Object[]>());
    when(specificDateScheduleDao.create(Mockito.anyObject()))
        .thenReturn(schedules.getSpecificDate().get(0));
    when(recurringScheduleDao.create(Mockito.anyObject()))
        .thenReturn(schedules.getRecurringSchedule().get(0));

    SynchronizeResult result = scheduleManager.synchronizeSchedules();

    assertEquals("It should create schedules", result, new SynchronizeResult(1, 0, 0));

    this.assertCreateSchedules(
        schedules,
        schedules.getSpecificDate().get(0),
        schedules.getRecurringSchedule().get(0),
        noOfSpecificDateSchedules,
        noOfDOMRecurringSchedules,
        noOfDOWRecurringSchedules);
  }

  @Test
  public void testSynchronizeSchedules_with_no_policy_and_existed_schedules() {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String guid = TestDataSetupHelper.generateGuid();
    int noOfSpecificDateSchedules = 3;
    int noOfDOMRecurringSchedules = 3;
    int noOfDOWRecurringSchedules = 3;

    List<RecurringScheduleEntity> recurringEntities =
        new RecurringScheduleEntitiesBuilder(noOfDOMRecurringSchedules, noOfDOWRecurringSchedules)
            .setAppId(appId)
            .setGuid(guid)
            .setTimeZone(TestDataSetupHelper.timeZone)
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .setScheduleId()
            .build();
    List<SpecificDateScheduleEntity> specificDateEntities =
        new SpecificDateScheduleEntitiesBuilder(noOfSpecificDateSchedules)
            .setAppid(appId)
            .setGuid(guid)
            .setTimeZone(TestDataSetupHelper.timeZone)
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .setScheduleId()
            .build();
    Schedules schedules =
        new ScheduleBuilder()
            .setSpecificDate(specificDateEntities)
            .setRecurringSchedule(recurringEntities)
            .setTimeZone(TestDataSetupHelper.timeZone)
            .build();
    String[] appIdAndGuid = {appId, guid};

    when(policyJsonDao.getAllPolicies()).thenReturn(new ArrayList<PolicyJsonEntity>());
    when(specificDateScheduleDao.getDistinctAppIdAndGuidList())
        .thenReturn(
            new ArrayList() {
              {
                add(appIdAndGuid);
              }
            });
    when(recurringScheduleDao.getDistinctAppIdAndGuidList())
        .thenReturn(
            new ArrayList() {
              {
                add(appIdAndGuid);
              }
            });
    when(specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(appId))
        .thenReturn(schedules.getSpecificDate());
    when(recurringScheduleDao.findAllRecurringSchedulesByAppId(appId))
        .thenReturn(schedules.getRecurringSchedule());

    SynchronizeResult result = scheduleManager.synchronizeSchedules();

    assertThat("It should delete the schedules", result, is(new SynchronizeResult(0, 0, 1)));

    this.assertDeleteSchedules(schedules);
  }

  @Test
  public void
      testSynchronizeSchedules_with_both_policy_with_schedules_and_schedules_existed_and_guid_are_different()
          throws JsonProcessingException {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String guid = TestDataSetupHelper.generateGuid();
    String anotherGuid = TestDataSetupHelper.generateGuid();
    int noOfSpecificDateSchedules = 3;
    int noOfDOMRecurringSchedules = 3;
    int noOfDOWRecurringSchedules = 3;

    List<RecurringScheduleEntity> recurringEntities =
        new RecurringScheduleEntitiesBuilder(noOfDOMRecurringSchedules, noOfDOWRecurringSchedules)
            .setAppId(appId)
            .setGuid(guid)
            .setTimeZone(TestDataSetupHelper.timeZone)
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .setScheduleId()
            .build();
    List<SpecificDateScheduleEntity> specificDateEntities =
        new SpecificDateScheduleEntitiesBuilder(noOfSpecificDateSchedules)
            .setAppid(appId)
            .setGuid(guid)
            .setTimeZone(TestDataSetupHelper.timeZone)
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .setScheduleId()
            .build();
    Schedules schedules =
        new ScheduleBuilder()
            .setSpecificDate(specificDateEntities)
            .setRecurringSchedule(recurringEntities)
            .setTimeZone(TestDataSetupHelper.timeZone)
            .build();

    List<RecurringScheduleEntity> anotherRecurringEntities =
        new RecurringScheduleEntitiesBuilder(noOfDOMRecurringSchedules, noOfDOWRecurringSchedules)
            .setAppId(appId)
            .setGuid(anotherGuid)
            .setTimeZone(TestDataSetupHelper.timeZone)
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .setScheduleId()
            .build();
    List<SpecificDateScheduleEntity> anotherSpecificDateEntities =
        new SpecificDateScheduleEntitiesBuilder(noOfSpecificDateSchedules)
            .setAppid(appId)
            .setGuid(anotherGuid)
            .setTimeZone(TestDataSetupHelper.timeZone)
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .setScheduleId()
            .build();
    Schedules anotherSchedules =
        new ScheduleBuilder()
            .setSpecificDate(anotherSpecificDateEntities)
            .setRecurringSchedule(anotherRecurringEntities)
            .setTimeZone(TestDataSetupHelper.timeZone)
            .build();
    ApplicationSchedules anotherApplicationSchedule =
        new ApplicationPolicyBuilder(1, 5).setSchedules(anotherSchedules).build();

    String[] appIdAndGuid = {appId, guid};

    List<PolicyJsonEntity> policyJsonList =
        new ArrayList<PolicyJsonEntity>() {
          {
            add(
                new PolicyJsonEntityBuilder(appId, anotherGuid, anotherApplicationSchedule)
                    .build());
          }
        };
    when(policyJsonDao.getAllPolicies()).thenReturn(policyJsonList);
    when(specificDateScheduleDao.getDistinctAppIdAndGuidList())
        .thenReturn(
            new ArrayList() {
              {
                add(appIdAndGuid);
              }
            });
    when(recurringScheduleDao.getDistinctAppIdAndGuidList())
        .thenReturn(
            new ArrayList() {
              {
                add(appIdAndGuid);
              }
            });
    when(specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(appId))
        .thenReturn(schedules.getSpecificDate());
    when(recurringScheduleDao.findAllRecurringSchedulesByAppId(appId))
        .thenReturn(schedules.getRecurringSchedule());
    when(specificDateScheduleDao.create(Mockito.anyObject()))
        .thenReturn(anotherSchedules.getSpecificDate().get(0));
    when(recurringScheduleDao.create(anyObject()))
        .thenReturn(anotherSchedules.getRecurringSchedule().get(0));

    SynchronizeResult result = scheduleManager.synchronizeSchedules();

    assertThat("It should update the shedules", result, is(new SynchronizeResult(0, 1, 0)));

    this.assertDeleteSchedules(schedules);
    this.assertCreateSchedules(
        anotherSchedules,
        anotherSchedules.getSpecificDate().get(0),
        anotherSchedules.getRecurringSchedule().get(0),
        noOfSpecificDateSchedules,
        noOfDOMRecurringSchedules,
        noOfDOWRecurringSchedules);
  }

  @Test
  public void testSynchronizeSchedules_with_both_policy_without_schedule_and_schedules_existed()
      throws JsonProcessingException {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String guid = TestDataSetupHelper.generateGuid();
    String anotherGuid = TestDataSetupHelper.generateGuid();
    int noOfSpecificDateSchedules = 3;
    int noOfDOMRecurringSchedules = 3;
    int noOfDOWRecurringSchedules = 3;

    List<RecurringScheduleEntity> recurringEntities =
        new RecurringScheduleEntitiesBuilder(noOfDOMRecurringSchedules, noOfDOWRecurringSchedules)
            .setAppId(appId)
            .setGuid(guid)
            .setTimeZone(TestDataSetupHelper.timeZone)
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .setScheduleId()
            .build();
    List<SpecificDateScheduleEntity> specificDateEntities =
        new SpecificDateScheduleEntitiesBuilder(noOfSpecificDateSchedules)
            .setAppid(appId)
            .setGuid(guid)
            .setTimeZone(TestDataSetupHelper.timeZone)
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .setScheduleId()
            .build();
    Schedules schedules =
        new ScheduleBuilder()
            .setSpecificDate(specificDateEntities)
            .setRecurringSchedule(recurringEntities)
            .setTimeZone(TestDataSetupHelper.timeZone)
            .build();

    ApplicationSchedules anotherApplicationSchedule =
        new ApplicationPolicyBuilder(1, 5).setSchedules(null).build();

    String[] appIdAndGuid = {appId, guid};

    List<PolicyJsonEntity> policyJsonList =
        new ArrayList<PolicyJsonEntity>() {
          {
            add(
                new PolicyJsonEntityBuilder(appId, anotherGuid, anotherApplicationSchedule)
                    .build());
          }
        };
    when(policyJsonDao.getAllPolicies()).thenReturn(policyJsonList);
    when(specificDateScheduleDao.getDistinctAppIdAndGuidList())
        .thenReturn(
            new ArrayList() {
              {
                add(appIdAndGuid);
              }
            });
    when(recurringScheduleDao.getDistinctAppIdAndGuidList())
        .thenReturn(
            new ArrayList() {
              {
                add(appIdAndGuid);
              }
            });
    when(specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(appId))
        .thenReturn(schedules.getSpecificDate());
    when(recurringScheduleDao.findAllRecurringSchedulesByAppId(appId))
        .thenReturn(schedules.getRecurringSchedule());

    SynchronizeResult result = scheduleManager.synchronizeSchedules();

    assertThat("It should update the shedules", result, is(new SynchronizeResult(0, 1, 0)));

    this.assertDeleteSchedules(schedules);
  }

  @Test
  public void
      testSynchronizeSchedules_with_both_policy_and_schedules_existed_and_guid_are_the_same()
          throws JsonProcessingException {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    String guid = TestDataSetupHelper.generateGuid();
    int noOfSpecificDateSchedules = 3;
    int noOfDOMRecurringSchedules = 3;
    int noOfDOWRecurringSchedules = 3;

    List<RecurringScheduleEntity> recurringEntities =
        new RecurringScheduleEntitiesBuilder(noOfDOMRecurringSchedules, noOfDOWRecurringSchedules)
            .setAppId(appId)
            .setGuid(guid)
            .setTimeZone(TestDataSetupHelper.timeZone)
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .setScheduleId()
            .build();
    List<SpecificDateScheduleEntity> specificDateEntities =
        new SpecificDateScheduleEntitiesBuilder(noOfSpecificDateSchedules)
            .setAppid(appId)
            .setGuid(guid)
            .setTimeZone(TestDataSetupHelper.timeZone)
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .setScheduleId()
            .build();
    Schedules schedules =
        new ScheduleBuilder()
            .setSpecificDate(specificDateEntities)
            .setRecurringSchedule(recurringEntities)
            .setTimeZone(TestDataSetupHelper.timeZone)
            .build();

    List<RecurringScheduleEntity> anotherRecurringEntities =
        new RecurringScheduleEntitiesBuilder(noOfDOMRecurringSchedules, noOfDOWRecurringSchedules)
            .setAppId(appId)
            .setGuid(guid)
            .setTimeZone(TestDataSetupHelper.timeZone)
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .setScheduleId()
            .build();
    List<SpecificDateScheduleEntity> anotherSpecificDateEntities =
        new SpecificDateScheduleEntitiesBuilder(noOfSpecificDateSchedules)
            .setAppid(appId)
            .setGuid(guid)
            .setTimeZone(TestDataSetupHelper.timeZone)
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .setScheduleId()
            .build();
    Schedules anotherSchedules =
        new ScheduleBuilder()
            .setSpecificDate(anotherSpecificDateEntities)
            .setRecurringSchedule(anotherRecurringEntities)
            .setTimeZone(TestDataSetupHelper.timeZone)
            .build();
    ApplicationSchedules anotherapplicationSchedule =
        new ApplicationPolicyBuilder(1, 5).setSchedules(anotherSchedules).build();

    String[] appIdAndGuid = {appId, guid};

    List<PolicyJsonEntity> policyJsonList =
        new ArrayList<PolicyJsonEntity>() {
          {
            add(new PolicyJsonEntityBuilder(appId, guid, anotherapplicationSchedule).build());
          }
        };
    when(policyJsonDao.getAllPolicies()).thenReturn(policyJsonList);
    when(specificDateScheduleDao.getDistinctAppIdAndGuidList())
        .thenReturn(
            new ArrayList() {
              {
                add(appIdAndGuid);
              }
            });
    when(recurringScheduleDao.getDistinctAppIdAndGuidList())
        .thenReturn(
            new ArrayList() {
              {
                add(appIdAndGuid);
              }
            });
    when(specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(appId))
        .thenReturn(schedules.getSpecificDate());
    when(recurringScheduleDao.findAllRecurringSchedulesByAppId(appId))
        .thenReturn(schedules.getRecurringSchedule());

    SynchronizeResult result = scheduleManager.synchronizeSchedules();

    assertThat(
        "It should not update or create schedule", result, is(new SynchronizeResult(0, 0, 0)));

    verify(recurringScheduleDao, never()).create(any());
    verify(specificDateScheduleDao, never()).create(any());
    verify(activeScheduleDao, never()).create(any());

    verify(recurringScheduleDao, never()).delete(any());
    verify(specificDateScheduleDao, never()).delete(any());
    verify(activeScheduleDao, never()).deleteActiveSchedulesByAppId(anyString());
    verify(scheduleJobManager, never()).createCronJob(any());
    verify(scheduleJobManager, never()).createSimpleJob(any());
    verify(scheduleJobManager, never()).deleteJob(anyString(), anyLong(), any());
  }

  @Captor ArgumentCaptor<RecurringScheduleEntity> recurringCaptor;
  @Captor ArgumentCaptor<SpecificDateScheduleEntity> specificDateCaptor;

  private void assertCreateSchedules(
      Schedules schedules,
      SpecificDateScheduleEntity specificDateScheduleEntity,
      RecurringScheduleEntity recurringScheduleEntity,
      int noOfSpecificDateSchedules,
      int noOfDOMRecurringSchedules,
      int noOfDOWRecurringSchedules) {

    if (schedules.getSpecificDate() != null) {
      Mockito.verify(specificDateScheduleDao, Mockito.times(noOfSpecificDateSchedules))
          .create(specificDateCaptor.capture());
      List<SpecificDateScheduleEntity> specificDateList = specificDateCaptor.getAllValues();
      for (SpecificDateScheduleEntity foundSpecificDateScheduleEntity :
          schedules.getSpecificDate()) {
        assert (specificDateList.contains(foundSpecificDateScheduleEntity));
      }
    }

    if (schedules.getRecurringSchedule() != null) {
      Mockito.verify(
              recurringScheduleDao,
              Mockito.times(noOfDOMRecurringSchedules + noOfDOWRecurringSchedules))
          .create(recurringCaptor.capture());
      List<RecurringScheduleEntity> recurringList = recurringCaptor.getAllValues();

      for (RecurringScheduleEntity foundRecurringScheduleEntity :
          schedules.getRecurringSchedule()) {
        assert (recurringList.contains(foundRecurringScheduleEntity));
      }
    }

    Mockito.verify(scheduleJobManager, Mockito.times(noOfSpecificDateSchedules))
        .createSimpleJob(specificDateScheduleEntity);
    Mockito.verify(
            scheduleJobManager,
            Mockito.times(noOfDOMRecurringSchedules + noOfDOWRecurringSchedules))
        .createCronJob(recurringScheduleEntity);
  }

  private void assertDeleteSchedules(Schedules schedules) {

    if (schedules.getSpecificDate() != null) {
      for (SpecificDateScheduleEntity foundSpecificDateScheduleEntity :
          schedules.getSpecificDate()) {
        verify(specificDateScheduleDao, times(1)).delete(foundSpecificDateScheduleEntity);
        verify(scheduleJobManager, times(1))
            .deleteJob(
                foundSpecificDateScheduleEntity.getAppId(),
                foundSpecificDateScheduleEntity.getId(),
                ScheduleTypeEnum.SPECIFIC_DATE);
      }
    }

    if (schedules.getRecurringSchedule() != null) {
      for (RecurringScheduleEntity foundRecurringScheduleEntity :
          schedules.getRecurringSchedule()) {
        verify(recurringScheduleDao, times(1)).delete(foundRecurringScheduleEntity);
        verify(scheduleJobManager, times(1))
            .deleteJob(
                foundRecurringScheduleEntity.getAppId(),
                foundRecurringScheduleEntity.getId(),
                ScheduleTypeEnum.RECURRING);
      }
    }
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
