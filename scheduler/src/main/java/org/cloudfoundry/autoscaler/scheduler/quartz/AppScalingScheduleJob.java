package org.cloudfoundry.autoscaler.scheduler.quartz;

import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.Logger;
import org.quartz.JobDataMap;
import org.quartz.JobExecutionContext;
import org.quartz.JobExecutionException;
import org.quartz.JobKey;
import org.springframework.scheduling.quartz.QuartzJobBean;
import org.springframework.stereotype.Component;
import org.springframework.web.context.support.SpringBeanAutowiringSupport;

/**
 * 
 *
 */
@Component
public class AppScalingScheduleJob extends QuartzJobBean {
	private Logger logger = LogManager.getLogger(this.getClass());

	@Override
	protected void executeInternal(JobExecutionContext context) throws JobExecutionException {

		SpringBeanAutowiringSupport.processInjectionBasedOnCurrentContext(this);
		logger.info("Scheduling job is executing for app scaling action");

		JobDataMap dataMap = context.getJobDetail().getJobDataMap();
		JobKey jobKey = context.getJobDetail().getKey();
		String appId = dataMap.getString("appId");
		Long scheduleId = dataMap.getLong("scheduleId");
		Object schedule = dataMap.get("schedule");

		logger.info(String.format("Job Key: %s, Application Id: %s, Schedule Id: %s, Schedule: %s", jobKey, appId,
				scheduleId, schedule));
	}
}
