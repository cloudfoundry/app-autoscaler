package org.cloudfoundry.autoscaler.scheduler.service;

import static org.junit.Assert.assertTrue;

import java.util.HashMap;
import java.util.Map;
import java.util.Set;

import org.apache.commons.logging.Log;
import org.apache.commons.logging.LogFactory;
import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.DataSetupHelper;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.quartz.JobDetail;
import org.quartz.JobKey;
import org.quartz.Scheduler;
import org.quartz.SchedulerException;
import org.quartz.impl.StdSchedulerFactory;
import org.quartz.impl.matchers.GroupMatcher;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.test.context.ContextConfiguration;
import org.springframework.test.context.junit4.SpringRunner;

/**
 * @author Fujitsu
 *
 */
@RunWith(SpringRunner.class)
@ContextConfiguration(locations = { "classpath:applicationContext-test.xml" })
public class ScalingJobManagerTest {

	@Autowired
	private ScalingJobManager scalingJobManager;
	private Log logger = LogFactory.getLog(this.getClass());

	@Before
	public void init() throws SchedulerException {
		// Clear previous schedules.
		Scheduler scheduler = StdSchedulerFactory.getDefaultScheduler();
		scheduler.clear();
	}

	@Test
	public void testCreateSimpleJob_01() throws Exception {
		logger.info("Executing Test Create Simple Job to create one schedule ...");
		ScheduleEntity scheduleEntity = DataSetupHelper.generateScheduleEntity();
		scheduleEntity.setScheduleId(Long.valueOf(1L));

		logger.info("=======  Create scheduling job for the schedule =======");
		scalingJobManager.createSimpleJob(scheduleEntity);

		assertJobCreated(scheduleEntity.getScheduleId());

		logger.info("======= Test Completed =======");
	}

	private void assertJobCreated(Long scheduleId) throws Exception, SchedulerException {
		Map<String, JobDetail> scheduleIdJobDetailMap = getSchedulerJobs();
		Set<String> jobKeys = scheduleIdJobDetailMap.keySet();

		logger.info("=======  Check there is a job for starting the scaling action =======");
		assertTrue(jobKeys.contains(scheduleId + "_start"));

		logger.info("=======  Check there is a job for ending the scaling action =======");
		assertTrue(jobKeys.contains(scheduleId + "_end"));

	}

	private Map<String, JobDetail> getSchedulerJobs() throws SchedulerException {
		Scheduler scheduler = StdSchedulerFactory.getDefaultScheduler();
		Map<String, JobDetail> scheduleIdJobDetailMap = new HashMap<String, JobDetail>();

		for (String groupName : scheduler.getJobGroupNames()) {

			for (JobKey jobKey : scheduler.getJobKeys(GroupMatcher.jobGroupEquals(groupName))) {
				scheduleIdJobDetailMap.put(jobKey.getName(), scheduler.getJobDetail(jobKey));

			}

		}

		return scheduleIdJobDetailMap;
	}
}
