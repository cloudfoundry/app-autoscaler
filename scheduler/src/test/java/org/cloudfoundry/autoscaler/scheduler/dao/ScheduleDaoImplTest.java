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

	private String appId = TestDataSetupHelper.getAppId_1();

	@After
	@Transactional
	public void removeAllRecordsFromDatabase() {
		for (ScheduleEntity entity : scheduleDao.findAllSchedulesByAppId(appId)) {
			scheduleDao.delete(entity);
		}
	}

	@Test
	@Transactional
	public void testFindAllSchedules_with_no_schedules() {
		List<ScheduleEntity> schedulesFound = findAllSchedules();
		assertSchedulesFoundEquals(0, schedulesFound);
	}

	@Test
	@Transactional
	public void testCreateAndFindSchedules() {
		// Pass the expected schedules.
		assertCreateAndFindSchedules(1);

		assertCreateAndFindSchedules(5);
	}

	private List<ScheduleEntity> createSchedules(int noOfSpecificDateSchedulesToSetUp) {
		List<ScheduleEntity> specificDateScheduleEntities = TestDataSetupHelper
				.generateSpecificDateScheduleEntities(appId, noOfSpecificDateSchedulesToSetUp);
		List<ScheduleEntity> returnValues = new ArrayList<ScheduleEntity>();
		for (ScheduleEntity scheduleEntity : specificDateScheduleEntities) {
			ScheduleEntity entity = scheduleDao.create(scheduleEntity);
			returnValues.add(entity);
		}

		return returnValues;
	}

	private List<ScheduleEntity> findAllSchedules() {
		return scheduleDao.findAllSchedulesByAppId(appId);
	}

	private void assertCreateAndFindSchedules(int expectedSchedulesTobeFound) {

		List<ScheduleEntity> savedSchedules = createSchedules(expectedSchedulesTobeFound);
		assertCreatedScheduleIdNotNull(savedSchedules);

		List<ScheduleEntity> schedulesFound = findAllSchedules();
		assertSchedulesFoundEquals(expectedSchedulesTobeFound, schedulesFound);

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
