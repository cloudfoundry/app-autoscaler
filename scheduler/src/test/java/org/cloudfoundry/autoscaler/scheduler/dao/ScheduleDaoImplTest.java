package org.cloudfoundry.autoscaler.scheduler.dao;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNotNull;

import java.util.ArrayList;
import java.util.List;

import javax.transaction.Transactional;

import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.junit.After;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.junit4.SpringRunner;

/**
 * 
 *
 */
@RunWith(SpringRunner.class)
@SpringBootTest
public class ScheduleDaoImplTest {

	@Autowired
	private ScheduleDao scheduleDao;

	private String[] multipleAppIds = TestDataSetupHelper.getAppIds();
	private String[] singleAppId = new String[] { TestDataSetupHelper.getAppId_1() };
	private String appId = TestDataSetupHelper.getAppId_1();

	@After
	@Transactional
	public void removeAllRecordsFromDatabase() {
		for (String appId : multipleAppIds) {
			for (ScheduleEntity entity : scheduleDao.findAllSchedulesByAppId(appId)) {
				scheduleDao.delete(entity);
			}
		}
	}

	@Test
	@Transactional
	public void testFindAllSchedules_with_no_schedules() {
		List<ScheduleEntity> schedulesFound = findAllSchedules(appId);
		assertSchedulesFoundEquals(0, schedulesFound);
	}

	@Test
	@Transactional
	public void testCreateAndFindSchedules() {
		// Pass the expected schedules.
		assertCreateAndFindSchedules(singleAppId, 1);
		assertCreateAndFindSchedules(singleAppId, 5);

		assertCreateAndFindSchedules(multipleAppIds, 1);
		assertCreateAndFindSchedules(multipleAppIds, 5);
	}

	private List<ScheduleEntity> createSchedules(String appId, int noOfSpecificDateSchedulesToSetUp) {
		List<ScheduleEntity> specificDateScheduleEntities = TestDataSetupHelper
				.generateSpecificDateScheduleEntities(appId, noOfSpecificDateSchedulesToSetUp);
		List<ScheduleEntity> returnValues = new ArrayList<ScheduleEntity>();
		for (ScheduleEntity scheduleEntity : specificDateScheduleEntities) {
			ScheduleEntity entity = scheduleDao.create(scheduleEntity);
			returnValues.add(entity);
		}

		return returnValues;
	}

	private List<ScheduleEntity> findAllSchedules(String appId) {
		return scheduleDao.findAllSchedulesByAppId(appId);
	}

	private void assertCreateAndFindSchedules(String[] appIds, int expectedSchedulesTobeFound) {

		for (String appId : appIds) {
			List<ScheduleEntity> savedSchedules = createSchedules(appId, expectedSchedulesTobeFound);
			assertCreatedScheduleIdNotNull(savedSchedules);
		}

		for (String appId : appIds) {
			List<ScheduleEntity> schedulesFound = findAllSchedules(appId);
			assertSchedulesFoundEquals(expectedSchedulesTobeFound, schedulesFound);
		}
		// reset all records for next test.
		removeAllRecordsFromDatabase();
	}

	private void assertSchedulesFoundEquals(int expectedSchedulesTobeFound, List<ScheduleEntity> schedulesFound) {
		assertEquals(expectedSchedulesTobeFound, schedulesFound.size());
	}

	private void assertCreatedScheduleIdNotNull(List<ScheduleEntity> specificDateScheduleEntities) {
		for (ScheduleEntity scheduleEntity : specificDateScheduleEntities) {
			assertNotNull(scheduleEntity.getId());
		}
	}

}
