package org.cloudfoundry.autoscaler.scheduler.quartz;

import org.apache.commons.logging.Log;
import org.apache.commons.logging.LogFactory;
import org.quartz.JobDataMap;
import org.quartz.JobExecutionContext;
import org.quartz.JobExecutionException;
import org.quartz.JobKey;
import org.springframework.scheduling.quartz.QuartzJobBean;
import org.springframework.stereotype.Component;
import org.springframework.web.context.support.SpringBeanAutowiringSupport;

/**
 * @author Fujitsu
 *
 */
@Component
public class AppScalingScheduleJob extends QuartzJobBean {
	private Log logger = LogFactory.getLog(this.getClass());

	@Override
	protected void executeInternal(JobExecutionContext context) throws JobExecutionException {

		SpringBeanAutowiringSupport.processInjectionBasedOnCurrentContext(this);
		logger.info("==========================================");
		logger.info("Scheduling job is executing for app scaling action");

		JobDataMap dataMap = context.getJobDetail().getJobDataMap();
		JobKey jobKey = context.getJobDetail().getKey();
		String appId = dataMap.getString("appId");
		Long scheduleId = dataMap.getLong("scheduleId");
		Object schedule = dataMap.get("schedule");

		logger.info("Job Key is " + jobKey);
		logger.info("Application Id: " + appId + " Schedule Id: " + scheduleId);
		logger.info("Schedule: " + schedule);
		logger.info("==========================================");

	}
}
