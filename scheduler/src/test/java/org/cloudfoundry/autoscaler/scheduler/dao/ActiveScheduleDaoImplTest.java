package org.cloudfoundry.autoscaler.scheduler.dao;

import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.Matchers.is;
import static org.hamcrest.Matchers.nullValue;

import jakarta.transaction.Transactional;
import java.util.List;
import org.cloudfoundry.autoscaler.scheduler.entity.ActiveScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataDbUtil;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.junit4.SpringRunner;

@RunWith(SpringRunner.class)
@SpringBootTest
@Transactional
public class ActiveScheduleDaoImplTest {

  @Autowired private ActiveScheduleDao activeScheduleDao;

  @Autowired private TestDataDbUtil testDataDbUtil;

  @Before
  public void before() {
    // Remove All ActiveSchedules
    testDataDbUtil.cleanupData();

    // Add fake test records.
    String appId = "appId_1";
    long scheduleId = 1L;
    long startJobIdentifier = 1L;
    testDataDbUtil.insertActiveSchedule(appId, scheduleId, 1, 5, 0, startJobIdentifier);

    appId = "appId_2";
    scheduleId = 2L;
    startJobIdentifier = 2L;
    testDataDbUtil.insertActiveSchedule(appId, scheduleId, 2, 7, 3, startJobIdentifier);
  }

  @Test
  public void testFindActiveSchedule_with_invalidId() {
    ActiveScheduleEntity activeScheduleEntity = activeScheduleDao.find(3L);
    assertThat("It should be null", activeScheduleEntity, nullValue());
  }

  @Test
  public void testCreateAndFindActiveSchedule() {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    Long scheduleId = 3L;
    ActiveScheduleEntity activeScheduleEntity =
        TestDataSetupHelper.generateActiveScheduleEntity(appId, scheduleId);

    assertThat(
        "It should have no active schedule",
        testDataDbUtil.getNumberOfActiveSchedulesByScheduleId(scheduleId),
        is(0L));

    activeScheduleDao.create(activeScheduleEntity);

    assertThat(
        "It should have one active schedule",
        testDataDbUtil.getNumberOfActiveSchedulesByScheduleId(scheduleId),
        is(1L));

    ActiveScheduleEntity foundActiveScheduleEntity =
        activeScheduleDao.find(activeScheduleEntity.getId());

    assertThat(
        "Both active schedules should be equal",
        activeScheduleEntity,
        is(foundActiveScheduleEntity));
  }

  @Test
  public void testDeleteActiveSchedule() {
    Long activeScheduleId = 2L;
    Long startJobIdentifier = 2L;

    assertThat(
        "It should have one active schedule",
        testDataDbUtil.getNumberOfActiveSchedulesByScheduleId(activeScheduleId),
        is(1L));

    int deletedActiveSchedules = activeScheduleDao.delete(activeScheduleId, startJobIdentifier);

    assertThat("It should be 1", deletedActiveSchedules, is(1));
    assertThat(
        "It should have no active schedule",
        testDataDbUtil.getNumberOfActiveSchedulesByScheduleId(activeScheduleId),
        is(0L));
  }

  @Test
  public void testDeleteActiveSchedule_with_nullValue() {

    assertThat("It should be 2", testDataDbUtil.getNumberOfActiveSchedules(), is(2L));

    int number = activeScheduleDao.delete(null, null);

    assertThat("It should be 0", number, is(0));
    assertThat("It should be 2", testDataDbUtil.getNumberOfActiveSchedules(), is(2L));
  }

  @Test
  public void testDeleteActiveSchedule_with_invalidId() {

    assertThat("It should be 2", testDataDbUtil.getNumberOfActiveSchedules(), is(2L));

    int deletedActiveSchedules = activeScheduleDao.delete(7L, 3L);

    assertThat("It should be 0", deletedActiveSchedules, is(0));
    assertThat("It should be 2", testDataDbUtil.getNumberOfActiveSchedules(), is(2L));
  }

  @Test
  public void testDeleteAllActiveSchedulesByAppId() {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    long scheduleId = 3L;
    ActiveScheduleEntity activeScheduleEntity =
        TestDataSetupHelper.generateActiveScheduleEntity(appId, scheduleId);
    activeScheduleDao.create(activeScheduleEntity);

    scheduleId = 4L;
    activeScheduleEntity = TestDataSetupHelper.generateActiveScheduleEntity(appId, scheduleId);
    activeScheduleDao.create(activeScheduleEntity);

    scheduleId = 5L;
    activeScheduleEntity = TestDataSetupHelper.generateActiveScheduleEntity(appId, scheduleId);
    activeScheduleDao.create(activeScheduleEntity);

    assertThat(
        "It should have 3 active schedules",
        testDataDbUtil.getNumberOfActiveSchedulesByAppId(appId),
        is(3L));

    activeScheduleDao.deleteActiveSchedulesByAppId(appId);

    assertThat(
        "It should have no active schedules",
        testDataDbUtil.getNumberOfActiveSchedulesByAppId(appId),
        is(0L));
  }

  @Test
  public void testFindActiveScheduleByAppId() {
    String appId = "appId_1";
    List<ActiveScheduleEntity> activeScheduleEntities = activeScheduleDao.findByAppId(appId);
    assertThat("It should have one active schedule", activeScheduleEntities.size(), is(1));
  }

  @Test
  public void testFindActiveScheduleByAppId_with_invalidAppId() {
    String appId = "invalid_appId";
    List<ActiveScheduleEntity> activeScheduleEntities = activeScheduleDao.findByAppId(appId);
    assertThat("It should have no active schedule", activeScheduleEntities.size(), is(0));
  }
}
