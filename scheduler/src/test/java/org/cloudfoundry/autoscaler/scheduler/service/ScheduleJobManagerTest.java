package org.cloudfoundry.autoscaler.scheduler.service;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertTrue;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.TimeUnit;

import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.JobActionEnum;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.ValidationErrorResult;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.quartz.JobDetail;
import org.quartz.JobKey;
import org.quartz.Scheduler;
import org.quartz.SchedulerException;
import org.quartz.impl.matchers.GroupMatcher;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.annotation.DirtiesContext;
import org.springframework.test.annotation.DirtiesContext.ClassMode;
import org.springframework.test.context.junit4.SpringRunner;

/**
 * 
 *
 */
@RunWith(SpringRunner.class)
@SpringBootTest
@DirtiesContext(classMode = ClassMode.BEFORE_EACH_TEST_METHOD)
public class ScheduleJobManagerTest {

	@Autowired
	private Scheduler scheduler;

	@Autowired
	private ScheduleJobManager scalingJobManager;

	@Autowired
	private ValidationErrorResult validationErrorResult;

	private String appId = TestDataSetupHelper.getAppId_1();

	@Before
	public void init() throws SchedulerException {
		// Clear previous schedules.
		validationErrorResult.initEmpty();
		scheduler.clear();
	}

	@Test
	public void testCreateSimpleJob_01() throws Exception {
		// Create jobs for one schedule
		List<ScheduleEntity> specificDateScheduleEntities = TestDataSetupHelper
				.generateSpecificDateScheduleEntitiesWithCurrentStartEndTime(appId, 1);

		assertEquals(1, specificDateScheduleEntities.size());

		ScheduleEntity scheduleEntity = specificDateScheduleEntities.get(0);
		scheduleEntity.setId(1L);
		scalingJobManager.createSimpleJob(scheduleEntity);
		Long scheduleId = scheduleEntity.getId();

		Map<String, JobDetail> scheduleIdJobDetailMap = getSchedulerJobs();
		Set<String> jobKeys = scheduleIdJobDetailMap.keySet();

		assertTrue(jobKeys.contains(ScheduleJobHelper.generateJobKey(scheduleId, JobActionEnum.START)));
		assertTrue(jobKeys.contains(ScheduleJobHelper.generateJobKey(scheduleId, JobActionEnum.END)));

		Thread.sleep(TimeUnit.SECONDS.toMillis(10));
		assertEquals("Expected number of jobs not started", scheduleId * 2,
				scheduler.getMetaData().getNumberOfJobsExecuted());

	}

	@Test
	public void testCreateSimpleJob_02() throws Exception {
		// Create jobs for two schedules
		int noOfSpecificDateSchedulesToSetUp = 2;
		List<ScheduleEntity> specificDateScheduleEntities = TestDataSetupHelper
				.generateSpecificDateScheduleEntitiesWithCurrentStartEndTime(appId, noOfSpecificDateSchedulesToSetUp);
		Long index = 0L;
		List<Long> scheduleIdList = new ArrayList<Long>();
		for (ScheduleEntity scheduleEntity : specificDateScheduleEntities) {
			Long scheduleId = ++index;
			scheduleEntity.setId(scheduleId);
			scalingJobManager.createSimpleJob(scheduleEntity);
			scheduleIdList.add(scheduleId);
		}

		Map<String, JobDetail> scheduleIdJobDetailMap = getSchedulerJobs();
		Set<String> jobKeys = scheduleIdJobDetailMap.keySet();
		for (Long scheduleId : scheduleIdList) {
			assertTrue(jobKeys.contains(ScheduleJobHelper.generateJobKey(scheduleId, JobActionEnum.START)));
			assertTrue(jobKeys.contains(ScheduleJobHelper.generateJobKey(scheduleId, JobActionEnum.END)));
		}

		Thread.sleep(TimeUnit.SECONDS.toMillis(10));
		assertEquals("Expected number of jobs not started", noOfSpecificDateSchedulesToSetUp * 2,
				scheduler.getMetaData().getNumberOfJobsExecuted());

	}

	private Map<String, JobDetail> getSchedulerJobs() throws SchedulerException {
		Map<String, JobDetail> scheduleIdJobDetailMap = new HashMap<>();

		for (String groupName : scheduler.getJobGroupNames()) {

			for (JobKey jobKey : scheduler.getJobKeys(GroupMatcher.jobGroupEquals(groupName))) {
				scheduleIdJobDetailMap.put(jobKey.getName(), scheduler.getJobDetail(jobKey));

			}

		}

		return scheduleIdJobDetailMap;
	}
}
