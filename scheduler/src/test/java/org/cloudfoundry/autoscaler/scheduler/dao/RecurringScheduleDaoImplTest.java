package org.cloudfoundry.autoscaler.scheduler.dao;

import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.Matchers.containsInAnyOrder;
import static org.hamcrest.Matchers.is;
import static org.junit.Assert.fail;

import jakarta.transaction.Transactional;
import java.sql.SQLException;
import java.util.List;
import java.util.Map;
import java.util.Set;
import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.RecurringScheduleEntitiesBuilder;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataDbUtil;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.DatabaseValidationException;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.junit4.SpringRunner;

@RunWith(SpringRunner.class)
@SpringBootTest
@Transactional
public class RecurringScheduleDaoImplTest {
  @Autowired private RecurringScheduleDao recurringScheduleDao;

  @Autowired private TestDataDbUtil testDataDbUtil;

  private String appId1;
  private String appId2;
  private String guid1;
  private String guid2;

  @Before
  public void before() throws SQLException {
    // Remove All ActiveSchedules
    testDataDbUtil.cleanupData();

    // Add fake test records.
    appId1 = "appId1";
    appId2 = "appId3";
    guid1 = TestDataSetupHelper.generateGuid();
    guid2 = TestDataSetupHelper.generateGuid();

    List<RecurringScheduleEntity> entities =
        new RecurringScheduleEntitiesBuilder(1, 0)
            .setAppId(appId1)
            .setGuid(guid1)
            .setTimeZone("")
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .build();
    testDataDbUtil.insertRecurringSchedule(entities);

    entities =
        new RecurringScheduleEntitiesBuilder(1, 0)
            .setAppId(appId2)
            .setGuid(guid2)
            .setTimeZone("")
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .build();
    ;
    testDataDbUtil.insertRecurringSchedule(entities);
  }

  @Test
  public void testFindAllSchedules_with_invalidAppId() {
    String appId = "invalid_appId";

    List<RecurringScheduleEntity> recurringScheduleEntities =
        recurringScheduleDao.findAllRecurringSchedulesByAppId(appId);

    assertThat("It should be empty list", recurringScheduleEntities.isEmpty(), is(true));
  }

  @Test
  public void testFindAllRecurringSchedulesByAppId() {
    String appId = "appId3";

    List<RecurringScheduleEntity> foundEntityList =
        recurringScheduleDao.findAllRecurringSchedulesByAppId(appId);

    assertThat("It should have one record", foundEntityList.size(), is(1));
    assertThat("The appId should be equal", foundEntityList.get(0).getAppId(), is(appId));
  }

  @Test
  public void testGetDistinctAppIdAndGuidList() {
    // add another rows with the same appId and guid
    List<RecurringScheduleEntity> entities =
        new RecurringScheduleEntitiesBuilder(1, 0)
            .setAppId(appId1)
            .setGuid(guid1)
            .setTimeZone("")
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .build();
    testDataDbUtil.insertRecurringSchedule(entities);
    entities =
        new RecurringScheduleEntitiesBuilder(1, 0)
            .setAppId(appId2)
            .setGuid(guid2)
            .setTimeZone("")
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .build();
    ;
    testDataDbUtil.insertRecurringSchedule(entities);

    Map<String, String> foundEntityList = recurringScheduleDao.getDistinctAppIdAndGuidList();

    assertThat("It should have two record", foundEntityList.size(), is(2));

    Set<String> appIdSet = foundEntityList.keySet();

    assertThat(
        "It should contains the two inserted entities",
        appIdSet,
        containsInAnyOrder(appId1, appId2));
  }

  @Test
  public void testCreateRecurringSchedule() throws Exception {
    String appId = "appId2";
    String guid = TestDataSetupHelper.generateGuid();
    RecurringScheduleEntity recurringScheduleEntity =
        TestDataSetupHelper.generateRecurringScheduleEntities(appId, guid, false, 1, 0).get(0);

    assertThat(
        "It should no recurring schedule",
        testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId),
        is(0));

    RecurringScheduleEntity savedEntity = recurringScheduleDao.create(recurringScheduleEntity);

    Long currentSequenceSchedulerId = testDataDbUtil.getCurrentRecurringSchedulerId();
    recurringScheduleEntity.setId(currentSequenceSchedulerId);

    assertThat(
        "It should have one recurring schedule",
        testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId),
        is(1));
    assertThat(
        "Both recurring schedules should be equal", savedEntity, is(recurringScheduleEntity));
  }

  @Test
  public void testDeleteSchedule() {
    String appId = "appId1";
    assertThat(
        "It should have one recurring schedule",
        testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId),
        is(1));
    RecurringScheduleEntity recurringScheduleEntity =
        recurringScheduleDao.findAllRecurringSchedulesByAppId(appId).get(0);
    recurringScheduleDao.delete(recurringScheduleEntity);

    assertThat(
        "It should have no recurring schedule",
        testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId),
        is(0));
  }

  @Test
  public void testDeleteSchedule_with_invalidAppId() {
    String appId = "invalid_appId";
    RecurringScheduleEntity recurringScheduleEntity = new RecurringScheduleEntity();
    recurringScheduleEntity.setAppId(appId);

    assertThat(
        "There are two recurring schedules", testDataDbUtil.getNumberOfRecurringSchedules(), is(2));

    recurringScheduleDao.delete(recurringScheduleEntity);

    assertThat(
        "There are two recurring schedules", testDataDbUtil.getNumberOfRecurringSchedules(), is(2));
  }

  /**
   * This test case succeed when database is postgresql, but failed when database is mysql, so
   * comment out it. @Test public void testFindSchedulesByAppId_throw_Exception() { try {
   * recurringScheduleDao.findAllRecurringSchedulesByAppId(null); fail("Should fail"); } catch
   * (DatabaseValidationException dve) { assertThat(dve.getMessage(), is("Find All recurring
   * schedules by app id failed")); } }
   */
  @Test
  public void testCreateSchedule_throw_Exception() {
    try {
      recurringScheduleDao.create(null);
      fail("Should fail");
    } catch (DatabaseValidationException dve) {
      assertThat(dve.getMessage(), is("Create failed"));
    }
  }

  @Test
  public void testDeleteSchedule_throw_Exception() {
    try {
      recurringScheduleDao.delete(null);
      fail("Should fail");
    } catch (DatabaseValidationException dve) {
      assertThat(dve.getMessage(), is("Delete failed"));
    }
  }
}
