package org.cloudfoundry.autoscaler.scheduler.dao;

import static org.hamcrest.Matchers.is;
import static org.junit.Assert.assertThat;
import static org.junit.Assert.fail;

import java.io.IOException;
import java.sql.SQLException;
import java.util.HashSet;
import java.util.List;
import java.util.Set;

import javax.transaction.Transactional;

import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.ConsulUtil;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataDbUtil;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.DatabaseValidationException;
import org.junit.AfterClass;
import org.junit.Before;
import org.junit.BeforeClass;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.junit4.SpringRunner;

@RunWith(SpringRunner.class)
@SpringBootTest
@Transactional
public class RecurringScheduleDaoImplTest {
	@Autowired
	private RecurringScheduleDao recurringScheduleDao;

	@Autowired
	private TestDataDbUtil testDataDbUtil;

	private static ConsulUtil consulUtil;

	private RecurringScheduleEntity entity1, entity2;

	@BeforeClass
	public static void beforeClass() throws IOException {
		consulUtil = new ConsulUtil();
		consulUtil.start();
	}

	@AfterClass
	public static void afterClass() throws IOException, InterruptedException {
		consulUtil.stop();
	}

	@Before
	public void before() throws SQLException {
		// Remove All ActiveSchedules
		testDataDbUtil.cleanupData();

		// Add fake test records.
		String appId = "appId1";
		String guid = TestDataSetupHelper.generateGuid();
		List<RecurringScheduleEntity> entities = TestDataSetupHelper.generateRecurringScheduleEntities(appId, guid,false, 1,
				0);
		testDataDbUtil.insertRecurringSchedule(entities);
		entity1 = entities.get(0);

		appId = "appId3";
		guid = TestDataSetupHelper.generateGuid();
		entities = TestDataSetupHelper.generateRecurringScheduleEntities(appId, guid, false, 0, 1);
		testDataDbUtil.insertRecurringSchedule(entities);
		entity2 = entities.get(0);
	}

	@Test
	public void testFindAllSchedules_with_invalidAppId() {
		String appId = "invalid_appId";

		List<RecurringScheduleEntity> recurringScheduleEntities = recurringScheduleDao
				.findAllRecurringSchedulesByAppId(appId);

		assertThat("It should be empty list", recurringScheduleEntities.isEmpty(), is(true));
	}

	@Test
	public void testFindAllRecurringSchedulesByAppId() {
		String appId = "appId3";

		List<RecurringScheduleEntity> foundEntityList = recurringScheduleDao.findAllRecurringSchedulesByAppId(appId);

		assertThat("It should have one record", foundEntityList.size(), is(1));
		assertThat("The appId should be equal", foundEntityList.get(0).getAppId(), is(appId));
	}

	@Test
	public void testFindAllRecurringSchedules() {
		String appId1 = "appId1";
		String appId3 = "appId3";
		List<RecurringScheduleEntity> foundEntityList = recurringScheduleDao.findAllRecurringSchedules();

		assertThat("It should have two record", foundEntityList.size(), is(2));
		Set<String> appIdSet = new HashSet<String>() {
			{
				add(foundEntityList.get(0).getAppId());
				add(foundEntityList.get(1).getAppId());
			}
		};
		assertThat("It should contains the two inserted entities",
				appIdSet.contains(appId1) && appIdSet.contains(appId3), is(true));

	}

	@Test
	public void testCreateRecurringSchedule() {
		String appId = "appId2";
		String guid = TestDataSetupHelper.generateGuid();
		RecurringScheduleEntity recurringScheduleEntity = TestDataSetupHelper
				.generateRecurringScheduleEntities(appId, guid, false, 1, 0).get(0);

		assertThat("It should no recurring schedule", testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId),
				is(0));

		RecurringScheduleEntity savedEntity = recurringScheduleDao.create(recurringScheduleEntity);

		Long currentSequenceSchedulerId = testDataDbUtil.getCurrentSequenceSchedulerId();
		recurringScheduleEntity.setId(currentSequenceSchedulerId);

		assertThat("It should have one recurring schedule", testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId),
				is(1));
		assertThat("Both recurring schedules should be equal", savedEntity, is(recurringScheduleEntity));
	}

	@Test
	public void testDeleteSchedule() {
		String appId = "appId1";
		assertThat("It should have one recurring schedule", testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId),
				is(1));
		RecurringScheduleEntity recurringScheduleEntity = recurringScheduleDao.findAllRecurringSchedulesByAppId(appId)
				.get(0);
		recurringScheduleDao.delete(recurringScheduleEntity);

		assertThat("It should have no recurring schedule", testDataDbUtil.getNumberOfRecurringSchedulesByAppId(appId),
				is(0));
	}

	@Test
	public void testDeleteSchedule_with_invalidAppId() {
		String appId = "invalid_appId";
		RecurringScheduleEntity recurringScheduleEntity = new RecurringScheduleEntity();
		recurringScheduleEntity.setAppId(appId);

		assertThat("There are two recurring schedules", testDataDbUtil.getNumberOfRecurringSchedules(), is(2));

		recurringScheduleDao.delete(recurringScheduleEntity);

		assertThat("There are two recurring schedules", testDataDbUtil.getNumberOfRecurringSchedules(), is(2));
	}

	@Test
	public void testFindSchedulesByAppId_throw_Exception() {
		try {
			recurringScheduleDao.findAllRecurringSchedulesByAppId(null);
			fail("Should fail");
		} catch (DatabaseValidationException dve) {
			assertThat(dve.getMessage(), is("Find All recurring schedules by app id failed"));
		}
	}

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
