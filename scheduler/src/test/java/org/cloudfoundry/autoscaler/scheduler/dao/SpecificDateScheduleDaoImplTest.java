package org.cloudfoundry.autoscaler.scheduler.dao;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNotNull;

import java.util.ArrayList;
import java.util.List;

import javax.transaction.Transactional;

import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
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
public class SpecificDateScheduleDaoImplTest {

	@Autowired
	private SpecificDateScheduleDao specificDateScheduleDao;

	@Before
	@Transactional
	public void removeAllRecordsFromDatabase() {
		List<String> allAppIds = TestDataSetupHelper.getAllGeneratedAppIds();
		for (String appId : allAppIds) {
			for (SpecificDateScheduleEntity entity : specificDateScheduleDao
					.findAllSpecificDateSchedulesByAppId(appId)) {
				specificDateScheduleDao.delete(entity);
			}
		}
	}

	@Test
	@Transactional
	public void testFindAllSchedules_withNoSchedules() {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		List<SpecificDateScheduleEntity> schedulesFound = findAllSchedules(appId);
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

	private List<SpecificDateScheduleEntity> createSchedules(String appId, int noOfSpecificDateSchedulesToSetUp) {
		List<SpecificDateScheduleEntity> specificDateScheduleEntities = TestDataSetupHelper
				.generateSpecificDateSchedules(appId, noOfSpecificDateSchedulesToSetUp, false);
		List<SpecificDateScheduleEntity> returnValues = new ArrayList<SpecificDateScheduleEntity>();
		for (SpecificDateScheduleEntity scheduleEntity : specificDateScheduleEntities) {
			SpecificDateScheduleEntity entity = specificDateScheduleDao.create(scheduleEntity);
			returnValues.add(entity);
		}

		return returnValues;
	}

	private List<SpecificDateScheduleEntity> findAllSchedules(String appId) {
		return specificDateScheduleDao.findAllSpecificDateSchedulesByAppId(appId);
	}

	private void assertCreateAndFindSchedules(int expectedSchedulesTobeFound, String... appIds) {

		for (String appId : appIds) {
			List<SpecificDateScheduleEntity> savedSchedules = createSchedules(appId, expectedSchedulesTobeFound);
			assertCreatedScheduleIdNotNull(savedSchedules);
		}

		for (String appId : appIds) {
			List<SpecificDateScheduleEntity> schedulesFound = findAllSchedules(appId);
			assertSchedulesFoundEquals(expectedSchedulesTobeFound, schedulesFound);
		}
	}

	private void assertSchedulesFoundEquals(int expectedSchedulesTobeFound,
			List<SpecificDateScheduleEntity> schedulesFound) {
		assertEquals(expectedSchedulesTobeFound, schedulesFound.size());
	}

	private void assertCreatedScheduleIdNotNull(List<SpecificDateScheduleEntity> specificDateScheduleEntities) {
		for (SpecificDateScheduleEntity scheduleEntity : specificDateScheduleEntities) {
			assertNotNull(scheduleEntity.getId());
		}
	}

}
