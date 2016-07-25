package org.cloudfoundry.autoscaler.scheduler.dao;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNotNull;
import static org.junit.Assert.assertTrue;

import java.util.ArrayList;
import java.util.List;

import javax.transaction.Transactional;

import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.junit4.SpringRunner;

@RunWith(SpringRunner.class)
@SpringBootTest
public class RecurringScheduleDaoImplTest {
	@Autowired
	private RecurringScheduleDao recurringScheduleDao;

	@Before
	@Transactional
	public void removeAllRecordsFromDatabase() {
		List<String> allAppIds = TestDataSetupHelper.getAllGeneratedAppIds();
		for (String appId : allAppIds) {
			for (RecurringScheduleEntity entity : recurringScheduleDao.findAllRecurringSchedulesByAppId(appId)) {
				recurringScheduleDao.delete(entity);
			}
		}
	}


	@Test
	@Transactional
	public void testFindAllSchedules_with_no_schedules() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		List<RecurringScheduleEntity> schedulesFound = findAllSchedules(appId);
		assertSchedulesFoundCountEquals(0, schedulesFound);
	}

	@Test
	@Transactional
	public void testCreateAndFindSchedules() {
		String[] allAppIds = TestDataSetupHelper.generateAppIds(5);
		// One recurring schedule for each app Id passed in the array
		assertCreateAndFindSchedules(allAppIds, 1);

		allAppIds = TestDataSetupHelper.generateAppIds(5);
		// Five recurring schedule for each app Id passed in the array
		assertCreateAndFindSchedules(allAppIds, 5);
	}

	private List<RecurringScheduleEntity> createSchedules(List<RecurringScheduleEntity> recurringScheduleEntities) {

		List<RecurringScheduleEntity> returnValues = new ArrayList<RecurringScheduleEntity>();
		for (RecurringScheduleEntity scheduleEntity : recurringScheduleEntities) {
			RecurringScheduleEntity entity = recurringScheduleDao.create(scheduleEntity);
			returnValues.add(entity);
		}

		return returnValues;
	}

	private List<RecurringScheduleEntity> findAllSchedules(String appId) {
		return recurringScheduleDao.findAllRecurringSchedulesByAppId(appId);
	}

	private void assertCreateAndFindSchedules(String[] appIds, int expectedSchedulesTobeFound) {
		int dowSchedules = expectedSchedulesTobeFound / 2;
		int domSchedules = expectedSchedulesTobeFound - dowSchedules;
		for (String appId : appIds) {
			List<RecurringScheduleEntity> recurringScheduleEntities = TestDataSetupHelper
					.generateRecurringSchedules(appId, dowSchedules, false);
			recurringScheduleEntities.addAll(
					TestDataSetupHelper.generateRecurringSchedules(appId, domSchedules, true));
			List<RecurringScheduleEntity> savedSchedules = createSchedules(recurringScheduleEntities);
			assertCreatedScheduleIdNotNull(savedSchedules);

			List<RecurringScheduleEntity> schedulesFound = findAllSchedules(appId);
			assertSchedulesFoundCountEquals(expectedSchedulesTobeFound, schedulesFound);

			assertSavedAndFoundEntitiesEquals(savedSchedules, schedulesFound);
		}
	}

	private void assertSavedAndFoundEntitiesEquals(List<RecurringScheduleEntity> savedSchedules,
			List<RecurringScheduleEntity> schedulesFound) {

		for (RecurringScheduleEntity recurringScheduleEntity : schedulesFound) {
			assertTrue(savedSchedules.contains(recurringScheduleEntity));
		}

	}

	private void assertSchedulesFoundCountEquals(int expectedSchedulesTobeFound,
			List<RecurringScheduleEntity> schedulesFound) {
		assertEquals(expectedSchedulesTobeFound, schedulesFound.size());
	}

	private void assertCreatedScheduleIdNotNull(List<RecurringScheduleEntity> recurringScheduleEntities) {
		for (RecurringScheduleEntity scheduleEntity : recurringScheduleEntities) {
			assertNotNull(scheduleEntity.getId());
		}
	}

}
