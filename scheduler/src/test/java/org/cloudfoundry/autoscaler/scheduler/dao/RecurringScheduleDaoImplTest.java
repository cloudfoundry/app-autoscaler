package org.cloudfoundry.autoscaler.scheduler.dao;

import static org.hamcrest.Matchers.is;
import static org.junit.Assert.assertThat;
import static org.junit.Assert.fail;

import java.io.IOException;
import java.sql.SQLException;
import java.util.List;

import javax.transaction.Transactional;

import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.ConsulUtil;
import org.cloudfoundry.autoscaler.scheduler.util.TestConfiguration;
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
public class RecurringScheduleDaoImplTest extends TestConfiguration {
	@Autowired
	private RecurringScheduleDao recurringScheduleDao;

	@Autowired
	private TestDataDbUtil testDataDbUtil;

	private static ConsulUtil consulUtil;

	@BeforeClass
	public static void beforeClass() throws IOException {
		consulUtil = new ConsulUtil();
		consulUtil.start();
	}

	@AfterClass
	public static void afterClass() throws IOException {
		consulUtil.stop();
	}

	@Before
	public void before() throws SQLException {
		// Remove All ActiveSchedules
		testDataDbUtil.cleanupData();

		// Add fake test records.
		String appId = "appId1";
		List<RecurringScheduleEntity> entities = TestDataSetupHelper.generateRecurringScheduleEntities(appId, 1, 0);
		testDataDbUtil.insertRecurringSchedule(entities);

		appId = "appId3";
		entities = TestDataSetupHelper.generateRecurringScheduleEntities(appId, 0, 1);
		testDataDbUtil.insertRecurringSchedule(entities);
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
	public void testCreateRecurringSchedule() {
		String appId = "appId2";

		RecurringScheduleEntity recurringScheduleEntity = TestDataSetupHelper
				.generateRecurringScheduleEntities(appId, 1, 0).get(0);

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
			assertThat(dve.getMessage(), is("Find All recurring schedules failed"));
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
