package org.cloudfoundry.autoscaler.scheduler.dao;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNotNull;

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
	public void afterTest() {
		for (ScheduleEntity entity : scheduleDao.findAllSchedulesByAppId(appId)) {
			scheduleDao.delete(entity);
		}
	}
	
	@Test
	@Transactional
	public void testFindAllSchedulesByAppId_01() {
		// Expected no schedule
		List<ScheduleEntity> schedulesFound = scheduleDao.findAllSchedulesByAppId(appId);
		assertEquals(0, schedulesFound.size());

	}
	
	@Test
	@Transactional
	public void testFindAllSchedulesByAppId_02() {
		// Expected one Schedule
		List<ScheduleEntity> specificDateScheduleEntities = TestDataSetupHelper.generateSpecificDateScheduleEntities(appId, 1);
		assertEquals(1, specificDateScheduleEntities.size());
		
		ScheduleEntity scheduleEntity = specificDateScheduleEntities.get(0);
		scheduleDao.create(scheduleEntity);
		
		List<ScheduleEntity>schedulesFound = scheduleDao.findAllSchedulesByAppId(appId);
		assertEquals(1, schedulesFound.size());

	}
	
	@Test
	@Transactional
	public void testFindAllSchedulesByAppId_03() {
		// Expected multiple Schedules
		int noOfSpecificDateSchedulesToSetUp = 4;
		List<ScheduleEntity> specificDateScheduleEntities = TestDataSetupHelper.generateSpecificDateScheduleEntities(appId, noOfSpecificDateSchedulesToSetUp);
		
		for (ScheduleEntity scheduleEntity: specificDateScheduleEntities) {
			scheduleDao.create(scheduleEntity);
		}
		
		scheduleDao.findAllSchedulesByAppId(appId);

		List<ScheduleEntity> schedulesFound = scheduleDao.findAllSchedulesByAppId(appId);
		assertEquals(noOfSpecificDateSchedulesToSetUp, schedulesFound.size());
	}
	
	@Test
	@Transactional
	public void testCreateSchedule_04() {
		// Create one schedule
		List<ScheduleEntity> specificDateScheduleEntities = TestDataSetupHelper.generateSpecificDateScheduleEntities(appId, 1);
		
		assertEquals(1, specificDateScheduleEntities.size());
		
		ScheduleEntity scheduleEntity = specificDateScheduleEntities.get(0);
		ScheduleEntity savedScheduleEntity = scheduleDao.create(scheduleEntity);
		assertNotNull(savedScheduleEntity.getId());

	}
	
	@Test
	@Transactional
	public void testCreateSchedule_05() {
		// Create multiple schedules
		int noOfSpecificDateSchedulesToSetUp = 4;
		List<ScheduleEntity> specificDateScheduleEntities = TestDataSetupHelper.generateSpecificDateScheduleEntities(appId, noOfSpecificDateSchedulesToSetUp);
		
		for (ScheduleEntity scheduleEntity: specificDateScheduleEntities) {
			scheduleDao.create(scheduleEntity);
			ScheduleEntity savedScheduleEntity = scheduleDao.create(scheduleEntity);
			assertNotNull(savedScheduleEntity.getId());
		}
		
	}
	
}
