package org.cloudfoundry.autoscaler.scheduler.dao;

import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.Matchers.containsInAnyOrder;
import static org.hamcrest.Matchers.is;
import static org.junit.Assert.fail;

import jakarta.transaction.Transactional;
import java.util.List;
import java.util.Map;
import java.util.Set;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.SpecificDateScheduleEntitiesBuilder;
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
public class SpecificDateScheduleDaoImplTest {

  @Autowired private SpecificDateScheduleDao specificDateScheduleDao;

  @Autowired private TestDataDbUtil testDataDbUtil;

  private String appId1;
  private String appId2;
  private String guid1;
  private String guid2;

  @Before
  public void before() {
    // Remove All ActiveSchedules
    testDataDbUtil.cleanupData();

    // Add fake test records.
    appId1 = "appId1";
    appId2 = "appId3";
    guid1 = TestDataSetupHelper.generateGuid();
    guid2 = TestDataSetupHelper.generateGuid();

    List<SpecificDateScheduleEntity> entities =
        new SpecificDateScheduleEntitiesBuilder(1)
            .setAppid(appId1)
            .setGuid(guid1)
            .setTimeZone("")
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .build();
    testDataDbUtil.insertSpecificDateSchedule(entities);

    entities =
        new SpecificDateScheduleEntitiesBuilder(1)
            .setAppid(appId2)
            .setGuid(guid2)
            .setTimeZone("")
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .build();
    testDataDbUtil.insertSpecificDateSchedule(entities);
  }

  @Test
  public void testFindAllSpecificDateSchedulesByAppId_with_invalidAppId() {
    String appId = "invalid_appId";

    List<SpecificDateScheduleEntity> specificDateScheduleEntities =
        specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(appId);

    assertThat("It should have empty list", specificDateScheduleEntities.isEmpty(), is(true));
  }

  @Test
  public void testFindAllSpecificDateSchedulesByAppId() {
    String appId = "appId1";

    List<SpecificDateScheduleEntity> foundEntityList =
        specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(appId);

    assertThat("It should have one specific date schedule", foundEntityList.size(), is(1));
    assertThat("The appId should be equal", foundEntityList.get(0).getAppId(), is(appId));
  }

  @Test
  public void testGetDistinctAppIdAndGuidList() {

    // add another rows with the same appId and guid
    List<SpecificDateScheduleEntity> entities =
        new SpecificDateScheduleEntitiesBuilder(1)
            .setAppid(appId1)
            .setGuid(guid1)
            .setTimeZone("")
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .build();
    testDataDbUtil.insertSpecificDateSchedule(entities);

    entities =
        new SpecificDateScheduleEntitiesBuilder(1)
            .setAppid(appId2)
            .setGuid(guid2)
            .setTimeZone("")
            .setDefaultInstanceMinCount(1)
            .setDefaultInstanceMaxCount(5)
            .build();
    testDataDbUtil.insertSpecificDateSchedule(entities);

    Map<String, String> foundEntityList = specificDateScheduleDao.getDistinctAppIdAndGuidList();

    assertThat("It should have two record", foundEntityList.size(), is(2));

    Set<String> appIdSet = foundEntityList.keySet();

    assertThat(
        "It should contains the two inserted entities",
        appIdSet,
        containsInAnyOrder(appId1, appId2));
  }

  @Test
  public void testCreateSpecificDateSchedule() throws Exception {
    String appId = "appId2";
    String guid = TestDataSetupHelper.generateGuid();
    SpecificDateScheduleEntity specificDateScheduleEntity =
        TestDataSetupHelper.generateSpecificDateScheduleEntities(appId, guid, false, 1).get(0);

    assertThat(
        "It should have no specific date schedule",
        testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId),
        is(0));

    SpecificDateScheduleEntity savedEntity =
        specificDateScheduleDao.create(specificDateScheduleEntity);

    Long currentSequenceSchedulerId = testDataDbUtil.getCurrentSpecificDateSchedulerId();
    specificDateScheduleEntity.setId(currentSequenceSchedulerId);

    assertThat(
        "It should have one specific date schedule",
        testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(appId),
        is(1));
    assertThat(
        "Both recurring schedules should be equal", savedEntity, is(specificDateScheduleEntity));
  }

  @Test
  public void testDeleteSchedule() {
    String appId = "appId1";

    assertThat(
        "It should have three records",
        testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(appId),
        is(1));

    SpecificDateScheduleEntity specificDateScheduleEntity =
        specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(appId).get(0);
    specificDateScheduleDao.delete(specificDateScheduleEntity);

    assertThat(
        "It should have three records",
        testDataDbUtil.getNumberOfSpecificDateSchedulesByAppId(appId),
        is(0));
  }

  @Test
  public void testDeleteSchedule_with_invalidAppId() {
    SpecificDateScheduleEntity specificDateScheduleEntity = new SpecificDateScheduleEntity();
    specificDateScheduleEntity.setAppId("invalid_appId");

    assertThat(
        "It should have three records", testDataDbUtil.getNumberOfSpecificDateSchedules(), is(2));

    specificDateScheduleDao.delete(specificDateScheduleEntity);

    assertThat(
        "It should have three records", testDataDbUtil.getNumberOfSpecificDateSchedules(), is(2));
  }

  /**
   * This test case succeed when database is postgresql, but failed when database is mysql, so
   * comment out it. @Test public void testFindSchedulesByAppId_throw_Exception() { try {
   * specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(null); fail("Should fail"); } catch
   * (DatabaseValidationException dve) { assertThat(dve.getMessage(), is("Find All specific date
   * schedules by app id failed")); } }
   */
  @Test
  public void testCreateSchedule_throw_Exception() {
    try {
      specificDateScheduleDao.create(null);
      fail("Should fail");
    } catch (DatabaseValidationException dve) {
      assertThat(dve.getMessage(), is("Create failed"));
    }
  }

  @Test
  public void testDeleteSchedule_throw_Exception() {
    try {
      specificDateScheduleDao.delete(null);
      fail("Should fail");
    } catch (DatabaseValidationException dve) {
      assertThat(dve.getMessage(), is("Delete failed"));
    }
  }
}
