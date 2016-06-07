package org.cloudfoundry.autoscaler.scheduler.quartz;

import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.Logger;
import org.quartz.JobDataMap;
import org.quartz.JobExecutionContext;
import org.quartz.JobExecutionException;
import org.springframework.scheduling.quartz.QuartzJobBean;
import org.springframework.stereotype.Component;
import org.springframework.web.context.support.SpringBeanAutowiringSupport;

/**
 * QuartzJobBean class that executes the job
 *
 */
@Component
public class AppScalingScheduleJob extends QuartzJobBean {
	private Logger logger = LogManager.getLogger(this.getClass());

	@Override
	protected void executeInternal(JobExecutionContext context) throws JobExecutionException {

		SpringBeanAutowiringSupport.processInjectionBasedOnCurrentContext(this);
		
		JobDataMap dataMap = context.getJobDetail().getJobDataMap();

		logger.info("Scheduling job is executing for app scaling action, Job Key: " + context.getJobDetail().getKey() 
				+ ", Application Id: " + dataMap.get("appId") 
				+ ", Schedule Id: " + dataMap.get("scheduleId") 
				+ ", Scaling Action: " + dataMap.get("scalingAction")
				+ ", Instance Min Count: " + dataMap.get("instanceMinCount")
				+ ", Instance Max Count: " + dataMap.get("instanceMaxCount"));

	}
}
