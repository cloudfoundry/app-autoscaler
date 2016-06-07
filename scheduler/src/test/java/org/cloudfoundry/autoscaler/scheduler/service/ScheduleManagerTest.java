package org.cloudfoundry.autoscaler.scheduler.service;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNotNull;

import org.apache.commons.logging.Log;
import org.apache.commons.logging.LogFactory;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationScalingSchedules;
import org.cloudfoundry.autoscaler.scheduler.util.DataSetupHelper;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.quartz.Scheduler;
import org.quartz.SchedulerException;
import org.quartz.impl.StdSchedulerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.test.context.ContextConfiguration;
import org.springframework.test.context.junit4.SpringRunner;

/**
 * @author Fujitsu
 *
 */
@RunWith(SpringRunner.class)
@ContextConfiguration(locations = { "classpath:applicationContext-test.xml" })
public class ScheduleManagerTest {

	@Autowired
	private ScalingScheduleManager scalingScheduleManager;
	private Log logger = LogFactory.getLog(this.getClass());

	@Before
	public void init() throws SchedulerException {
		// Clear previous schedules.
		Scheduler scheduler = StdSchedulerFactory.getDefaultScheduler();
		scheduler.clear();
	}

	@Test
	public void testCreateSchedules_01() throws Exception {
		logger.info("Executing Test Create Schedule to create one schedule...");

		int noOfSchedules = 1;
		ApplicationScalingSchedules schedules = DataSetupHelper.generateSchedules(noOfSchedules);
		String appId = scalingScheduleManager.createSchedules(schedules);

		logger.info("======= Check the application id is not null =======");
		assertNotNull(appId);

		logger.info(
				"======= Check the supplied application id is same as application id of the persisted schedule  =======");
		assertEquals(schedules.getApp_id(), appId);

		logger.info("======= Test Completed =======");
	}
}
