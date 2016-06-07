package org.cloudfoundry.autoscaler.scheduler.dao;

import static org.junit.Assert.assertNotNull;

import javax.transaction.Transactional;

import org.apache.commons.logging.Log;
import org.apache.commons.logging.LogFactory;
import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.DataSetupHelper;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.test.annotation.Rollback;
import org.springframework.test.context.ContextConfiguration;
import org.springframework.test.context.junit4.SpringRunner;

/**
 * @author Fujitsu
 *
 */
@RunWith(SpringRunner.class)
@ContextConfiguration(locations = { "classpath:applicationContext-test.xml" })
@Rollback(true)

public class ScheduleDaoImplTest {

	@Autowired
	ScheduleDao scheduleDao;
	private Log logger = LogFactory.getLog(this.getClass());

	@Test
	@Transactional
	public void testCreateSchedule_01() {
		logger.info("Executing Test Create Schedule to create one schedule ...");
		ScheduleEntity entity = DataSetupHelper.generateScheduleEntity();

		logger.info("=======  Create Schedule =======");
		ScheduleEntity result = scheduleDao.create(entity);

		logger.info("=======  Check the scheduleId is not null in the persisted entity =======");
		assertNotNull(result.getScheduleId());

		logger.info("======= Test Completed =======");

	}
}
