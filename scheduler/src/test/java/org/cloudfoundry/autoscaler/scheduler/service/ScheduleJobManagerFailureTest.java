package org.cloudfoundry.autoscaler.scheduler.service;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertTrue;

import java.util.List;

import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.ValidationErrorResult;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.Mockito;
import org.quartz.Scheduler;
import org.quartz.SchedulerException;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.annotation.DirtiesContext;
import org.springframework.test.context.ActiveProfiles;
import org.springframework.test.context.junit4.SpringRunner;

@ActiveProfiles("SchedulerMock")
@RunWith(SpringRunner.class)
@SpringBootTest
@DirtiesContext(classMode = DirtiesContext.ClassMode.AFTER_EACH_TEST_METHOD)
public class ScheduleJobManagerFailureTest {

	@Autowired
	private Scheduler scheduler;

	@Autowired
	private ScheduleJobManager scalingJobManager;

	@Autowired
	private ValidationErrorResult validationErrorResult;

	@Autowired
	MessageBundleResourceHelper messageBundleResourceHelper;

	private String appId = TestDataSetupHelper.getAppId_1();

	@Before
	public void init() throws SchedulerException {
		// Clear previous schedules.
		validationErrorResult.initEmpty();
		scheduler.clear();
	}

	@Test
	public void testCreateSimpleJob_Failure_with_throw_SchedulerException_at_Quartz() throws SchedulerException {
		
		// Set mock object for Quartz.
		Mockito.when(scheduler.scheduleJob(Mockito.anyObject(), Mockito.anyObject()))
				.thenThrow(new SchedulerException());
		int noOfSpecificDateSchedulesToSetUp = 1;
		
		List<ScheduleEntity> specificDateScheduleEntities = TestDataSetupHelper.generateSpecificDateScheduleEntities(appId, noOfSpecificDateSchedulesToSetUp);
		Long index = 0L;
		for (ScheduleEntity scheduleEntity: specificDateScheduleEntities) {
			scheduleEntity.setId(++index);
			scalingJobManager.createSimpleJob(scheduleEntity);
		}

		assertTrue("This test should have an Error.", validationErrorResult.hasErrors());
		List<String> errors = validationErrorResult.getAllErrorMessages();
		assertEquals(1, errors.size());

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.scheduler.error.create.failed",
				"app_id=" + appId);

		assertEquals(errorMessage, errors.get(0));
	}
}
