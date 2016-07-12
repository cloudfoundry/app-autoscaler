package org.cloudfoundry.autoscaler.scheduler.service;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNotNull;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.JobActionEnum;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.ValidationErrorResult;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.quartz.JobDataMap;
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
	public void initializer() throws SchedulerException {
		// Clear previous schedules.
		validationErrorResult.initEmpty();
		scheduler.clear();
	}

	@Test
	public void testCreateAndFindSimpleJobs() throws Exception {
		// Pass the expected schedules
		assertCreateAndFindSimpleJobs(1);
		assertCreateAndFindSimpleJobs(4);
	}

	private void assertCreateAndFindSimpleJobs(int expectedJobsTobeFound)
			throws SchedulerException, InterruptedException {
		// reset all records for this test.
		initializer();

		List<ScheduleEntity> scheduleEntities = createSimpleJob(expectedJobsTobeFound);
		assertCreatedJobs(scheduleEntities);
	}

	private void assertCreatedJobs(List<ScheduleEntity> scheduleEntities) throws SchedulerException {
		Map<String, JobDetail> scheduleIdJobDetailMap = getSchedulerJobs();

		for (ScheduleEntity entity : scheduleEntities) {
			JobDetail jobDetail = scheduleIdJobDetailMap
					.get(ScheduleJobHelper.generateJobKey(entity.getId(), JobActionEnum.START));
			assertJobDetails(entity, entity.getInstanceMinCount(), entity.getInstanceMaxCount(), JobActionEnum.START,
					jobDetail);

			jobDetail = scheduleIdJobDetailMap.get(ScheduleJobHelper.generateJobKey(entity.getId(), JobActionEnum.END));
			assertJobDetails(entity, entity.getDefaultInstanceMinCount(), entity.getDefaultInstanceMaxCount(),
					JobActionEnum.END, jobDetail);
		}
	}

	private void assertJobDetails(ScheduleEntity expectedEntity, int expectedInstanceMinCount,
			int expectedInstanceMaxCount, JobActionEnum expectedJobAction, JobDetail jobDetail) {
		assertNotNull("Expected existing jobDetail", jobDetail);
		JobDataMap map = jobDetail.getJobDataMap();
		assertEquals(expectedEntity.getAppId(), map.get("appId"));
		assertEquals(expectedEntity.getId(), map.get("scheduleId"));
		assertEquals(expectedJobAction, map.get("scalingAction"));
		assertEquals(expectedInstanceMinCount, map.get("instanceMinCount"));
		assertEquals(expectedInstanceMaxCount, map.get("instanceMaxCount"));
	}

	private List<ScheduleEntity> createSimpleJob(int expectedJobsTobeFound) {
		List<ScheduleEntity> specificDateScheduleEntities = TestDataSetupHelper
				.generateSpecificDateScheduleEntitiesWithCurrentStartEndTime(appId, expectedJobsTobeFound);
		Long index = 0L;
		for (ScheduleEntity scheduleEntity : specificDateScheduleEntities) {
			Long scheduleId = ++index;
			scheduleEntity.setId(scheduleId);
			scalingJobManager.createSimpleJob(scheduleEntity);
		}
		return specificDateScheduleEntities;
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
