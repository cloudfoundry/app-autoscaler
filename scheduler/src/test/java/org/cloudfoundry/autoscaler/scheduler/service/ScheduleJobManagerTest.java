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
		// Pass the expected schedules
		assertCreateAndFindSimpleJobs(4);
	}

	@Test
	public void testCreateSimpleJob_with_throw_SchedulerException_at_Quartz() throws SchedulerException {

		// Set mock object for Quartz.
		Mockito.doThrow(SchedulerException.class).when(scheduler).scheduleJob(Mockito.anyObject(), Mockito.anyObject());
		int noOfSpecificDateSchedulesToSetUp = 1;
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		List<SpecificDateScheduleEntity> specificDateScheduleEntities = TestDataSetupHelper
				.generateSpecificDateSchedules(appId, noOfSpecificDateSchedulesToSetUp, false);
		Long index = 0L;
		for (SpecificDateScheduleEntity scheduleEntity : specificDateScheduleEntities) {
			scheduleEntity.setId(++index);
			scalingJobManager.createSimpleJob(scheduleEntity);
		}

		assertTrue("This test should have an Error.", validationErrorResult.hasErrors());
		List<String> errors = validationErrorResult.getAllErrorMessages();
		assertEquals(1, errors.size());

		String errorMessage = messageBundleResourceHelper.lookupMessage("scheduler.error.create.failed",
				"app_id=" + appId);

		assertEquals(errorMessage, errors.get(0));
	}

	private void assertCreateAndFindSimpleJobs(int expectedJobsTobeFound)
			throws SchedulerException, InterruptedException {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		List<SpecificDateScheduleEntity> specificDateScheduleEntities = TestDataSetupHelper
				.generateSpecificDateSchedules(appId, expectedJobsTobeFound, true);

		createSimpleJob(specificDateScheduleEntities);
		assertScheduleJobMethodCallNum(expectedJobsTobeFound);
		assertCreatedSimpleJobs(specificDateScheduleEntities);
	}

	private void assertScheduleJobMethodCallNum(int expectedJobsTobeFound) throws SchedulerException {
		Mockito.verify(scheduler, Mockito.times(expectedJobsTobeFound * 2)).scheduleJob(Mockito.anyObject(),
				Mockito.anyObject());
	}

	private void assertCreatedSimpleJobs(List<SpecificDateScheduleEntity> scheduleEntities) throws SchedulerException {
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

	private void createSimpleJob(List<SpecificDateScheduleEntity> specificDateScheduleEntities) {
		Long index = 0L;
		for (SpecificDateScheduleEntity scheduleEntity : specificDateScheduleEntities) {
			Long scheduleId = ++index;
			scheduleEntity.setId(scheduleId);
			scalingJobManager.createSimpleJob(scheduleEntity);
		}
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
