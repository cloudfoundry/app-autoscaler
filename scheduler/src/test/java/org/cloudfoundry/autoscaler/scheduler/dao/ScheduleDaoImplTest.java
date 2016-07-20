package org.cloudfoundry.autoscaler.scheduler.dao;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNotNull;

import java.util.ArrayList;
import java.util.List;

import javax.transaction.Transactional;

import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.junit.Before;
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

	@Before
	@Transactional
	public void removeAllRecordsFromDatabase() {
		List<String> allAppIds = TestDataSetupHelper.getAllGeneratedAppIds();
		for (String appId : allAppIds) {
			for (ScheduleEntity entity : scheduleDao.findAllSchedulesByAppId(appId)) {
				scheduleDao.delete(entity);
			}
		}
	}

	@Test
	@Transactional
	public void testFindAllSchedules_withNoSchedules() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		List<ScheduleEntity> schedulesFound = findAllSchedules(appId);
		assertSchedulesFoundEquals(0, schedulesFound);
	}

	@Test
	@Transactional
	public void testCreateAndFindSchedules() {
		String[] allAppIds = TestDataSetupHelper.generateAppIds(5);
		// One specific schedule for each app Id passed in the array
		assertCreateAndFindSchedules(1, allAppIds);

		allAppIds = TestDataSetupHelper.generateAppIds(5);
		// Five specific schedule for each app Id passed in the array
		assertCreateAndFindSchedules(5, allAppIds);
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

	private void assertCreateAndFindSchedules(int expectedSchedulesTobeFound, String... appIds) {

		for (String appId : appIds) {
			List<ScheduleEntity> savedSchedules = createSchedules(appId, expectedSchedulesTobeFound);
			assertCreatedScheduleIdNotNull(savedSchedules);
		}

		for (String appId : appIds) {
			List<ScheduleEntity> schedulesFound = findAllSchedules(appId);
			assertSchedulesFoundEquals(expectedSchedulesTobeFound, schedulesFound);
		}
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
