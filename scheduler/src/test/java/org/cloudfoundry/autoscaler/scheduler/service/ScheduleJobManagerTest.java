package org.cloudfoundry.autoscaler.scheduler.service;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNotNull;
import static org.junit.Assert.assertTrue;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.JobActionEnum;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleTypeEnum;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.ValidationErrorResult;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.Mockito;
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
import org.springframework.test.context.ActiveProfiles;
import org.springframework.test.context.junit4.SpringRunner;

/**
 * 
 *
 */
@ActiveProfiles("SchedulerMock")
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

	@Autowired
	MessageBundleResourceHelper messageBundleResourceHelper;

	//	private String appId = TestDataSetupHelper.getAppId_1();

	@Before
	public void initializer() throws SchedulerException {
		// Clear previous schedules.
		scheduler.clear();
		Mockito.reset(scheduler);
	}

	@Test
	public void testCreateAndFindSimpleJobs() throws Exception {
		int schedulesToSetup = 4;
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		List<SpecificDateScheduleEntity> specificDateScheduleEntities = TestDataSetupHelper
				.generateSpecificDateSchedules(appId, schedulesToSetup, true);

		createSimpleJob(specificDateScheduleEntities);

		// The expected number of jobs would be twice the number of schedules( One job for start and one for end)
		int expectedJobsToBeCreated = 2 * schedulesToSetup;

		Map<JobKey, JobDetail> scheduleJobKeyDetailMap = getSchedulerJobs(
				ScheduleTypeEnum.SPECIFIC_DATE.getScheduleIdentifier());
		// Check expected jobs created
		assertEquals(expectedJobsToBeCreated, scheduleJobKeyDetailMap.size());

		for (ScheduleEntity scheduleEntity : specificDateScheduleEntities) {
			assertCreatedJobs(scheduleJobKeyDetailMap, scheduleEntity, ScheduleTypeEnum.SPECIFIC_DATE);
		}
	}

	@Test
	public void testCreateSimpleJob_with_throw_SchedulerException_at_Quartz() throws SchedulerException {

		// Set mock object for Quartz.
		Mockito.doThrow(SchedulerException.class).when(scheduler).scheduleJob(Mockito.anyObject(), Mockito.anyObject());
		int noOfSpecificDateSchedulesToSetUp = 1;
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		List<SpecificDateScheduleEntity> specificDateScheduleEntities = TestDataSetupHelper
				.generateSpecificDateSchedules(appId, noOfSpecificDateSchedulesToSetUp, false);
		createSimpleJob(specificDateScheduleEntities);

		assertTrue("This test should have an Error.", validationErrorResult.hasErrors());

		List<String> errors = validationErrorResult.getAllErrorMessages();
		assertEquals(1, errors.size());

		String errorMessage = messageBundleResourceHelper.lookupMessage("scheduler.error.create.failed",
				"app_id=" + appId);

		assertEquals(errorMessage, errors.get(0));
	}

	@Test
	public void testDeleteSimpleJobs() throws Exception {
		int schedulesToCreate = 2;
		int expectedJobsToBeCreated = 4; // 2 jobs per schedule, one for start and one for end
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		List<SpecificDateScheduleEntity> specificDateScheduleEntities = TestDataSetupHelper
				.generateSpecificDateSchedules(appId, schedulesToCreate, true);

		createSimpleJob(specificDateScheduleEntities);
		Map<JobKey, JobDetail> scheduleJobKeyDetailMap = getSchedulerJobs(
				ScheduleTypeEnum.SPECIFIC_DATE.getScheduleIdentifier());

		// Check expected jobs created
		assertEquals(expectedJobsToBeCreated, scheduleJobKeyDetailMap.size());

		// Delete the simple jobs
		for (ScheduleEntity scheduleEntity : specificDateScheduleEntities) {
			scalingJobManager.deleteJob(scheduleEntity.getAppId(), scheduleEntity.getId(),
					ScheduleTypeEnum.SPECIFIC_DATE);
		}

		scheduleJobKeyDetailMap = getSchedulerJobs(ScheduleTypeEnum.SPECIFIC_DATE.getScheduleIdentifier());
		// Check the jobs, the expected job count is 0.
		assertEquals(0, scheduleJobKeyDetailMap.size());

	}

	private void createSimpleJob(List<SpecificDateScheduleEntity> specificDateScheduleEntities) {
		Long index = 0L;
		for (SpecificDateScheduleEntity scheduleEntity : specificDateScheduleEntities) {
			Long scheduleId = ++index;
			scheduleEntity.setId(scheduleId);
			scalingJobManager.createSimpleJob(scheduleEntity);
		}
	}

	private void assertCreatedJobs(Map<JobKey, JobDetail> scheduleIdJobDetailMap, ScheduleEntity scheduleEntity,
			ScheduleTypeEnum scheduleType)
			throws SchedulerException {
		String appId = scheduleEntity.getAppId();
		Long scheduleId = scheduleEntity.getId();

		JobKey startJobKey = ScheduleJobHelper.generateJobKey(scheduleId, JobActionEnum.START,
				ScheduleTypeEnum.SPECIFIC_DATE);
		JobKey endJobKey = ScheduleJobHelper.generateJobKey(scheduleId, JobActionEnum.END,
				ScheduleTypeEnum.SPECIFIC_DATE);

		int instMinCount = scheduleEntity.getInstanceMinCount();
		int instMaxCount = scheduleEntity.getInstanceMaxCount();

		JobDetail jobDetail = scheduleIdJobDetailMap.get(startJobKey);
		assertJobDetails(appId, scheduleId, instMinCount, instMaxCount, JobActionEnum.START, jobDetail);

		instMinCount = scheduleEntity.getDefaultInstanceMinCount();
		instMaxCount = scheduleEntity.getDefaultInstanceMaxCount();
		jobDetail = scheduleIdJobDetailMap.get(endJobKey);
		assertJobDetails(appId, scheduleId, instMinCount, instMaxCount, JobActionEnum.END, jobDetail);
	}

	private void assertJobDetails(String expectedAppId, Long expectedScheduleId, int expectedInstanceMinCount,
			int expectedInstanceMaxCount, JobActionEnum expectedJobAction, JobDetail expectedJobDetail) {
		assertNotNull("Expected existing jobDetail", expectedJobDetail);
		JobDataMap map = expectedJobDetail.getJobDataMap();
		assertEquals(expectedAppId, map.get("appId"));
		assertEquals(expectedScheduleId, map.get("scheduleId"));
		assertEquals(expectedJobAction, map.get("scalingAction"));
		assertEquals(expectedInstanceMinCount, map.get("instanceMinCount"));
		assertEquals(expectedInstanceMaxCount, map.get("instanceMaxCount"));
	}

	private Map<JobKey, JobDetail> getSchedulerJobs(String groupName) throws SchedulerException {
		Map<JobKey, JobDetail> scheduleJobKeyDetailMap = new HashMap<>();

		for (JobKey jobKey : scheduler.getJobKeys(GroupMatcher.jobGroupEquals(groupName))) {
			scheduleJobKeyDetailMap.put(jobKey, scheduler.getJobDetail(jobKey));

		}

		return scheduleJobKeyDetailMap;
	}
	
}
