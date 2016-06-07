package org.cloudfoundry.autoscaler.scheduler.service;

import java.util.Date;
import java.util.TimeZone;

import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.quartz.AppScalingScheduleJob;
import org.cloudfoundry.autoscaler.scheduler.util.DateHelper;
import org.cloudfoundry.autoscaler.scheduler.util.ScalingActionEnum;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper;
import org.quartz.JobDataMap;
import org.quartz.JobDetail;
import org.quartz.Scheduler;
import org.quartz.SchedulerException;
import org.quartz.Trigger;
import org.quartz.impl.StdSchedulerFactory;
import org.springframework.stereotype.Service;

/**
 * Service class to persist the schedule entity in the database and create
 * scheduled job.
 * 
 * @author Fujitsu
 *
 */
@Service
public class ScalingJobManager {

	/**
	 * Creates simple job for the application scaling using helper methods. Here
	 * in two jobs are required, First job to tell the scaling decision maker
	 * scaling action needs to initiated Second job to tell the scaling decision
	 * maker scaling action needs to be ended.
	 * 
	 * @param scheduleEntity
	 * @throws Exception
	 */
	public void createSimpleJob(ScheduleEntity scheduleEntity) throws SchedulerException {

		Long scheduleId = scheduleEntity.getScheduleId();
		String jobStartId = scheduleId + "_start";
		String jobEndId = scheduleId + "_end";

		// Build the job
		JobDetail jobStartDetail = ScheduleJobHelper.buildJob(jobStartId, AppScalingScheduleJob.class);
		JobDetail jobEndDetail = ScheduleJobHelper.buildJob(jobEndId, AppScalingScheduleJob.class);

		// Set the data in JobDetail for starting the scaling job
		setupScalingScheduleJobData(jobStartDetail, scheduleEntity, ScalingActionEnum.START.getAction());
		// Set the data in JobDetail for ending the scaling job
		setupScalingScheduleJobData(jobEndDetail, scheduleEntity, ScalingActionEnum.END.getAction());

		// Build the trigger
		long triggerStartDateTime = scheduleEntity.getStartDate().getTime() + scheduleEntity.getStartTime().getTime();
		long triggerEndDateTime = scheduleEntity.getEndDate().getTime() + scheduleEntity.getEndTime().getTime();
		TimeZone policyTimeZone = TimeZone.getTimeZone(scheduleEntity.getTimezone());

		Date triggerStartDate = DateHelper.getDateWithZoneOffset(triggerStartDateTime, policyTimeZone);
		Date triggerEndDate = DateHelper.getDateWithZoneOffset(triggerEndDateTime, policyTimeZone);

		Trigger jobStartTrigger = ScheduleJobHelper.buildTrigger(jobStartId, jobStartDetail.getKey(), triggerStartDate);
		Trigger jobEndTrigger = ScheduleJobHelper.buildTrigger(jobEndId, jobEndDetail.getKey(), triggerEndDate);

		// Schedule the job
		Scheduler scheduler = new StdSchedulerFactory().getScheduler();
		scheduler.start();
		scheduler.scheduleJob(jobStartDetail, jobStartTrigger);
		scheduler.scheduleJob(jobEndDetail, jobEndTrigger);
	}

	/**
	 * Sets the data in the JobDetail object
	 * 
	 * @param jobDetail
	 * @param scheduleEntity
	 */
	private void setupScalingScheduleJobData(JobDetail jobDetail, ScheduleEntity scheduleEntity, String scalingAction) {

		JobDataMap jobDataMap = jobDetail.getJobDataMap();
		jobDataMap.put("appId", scheduleEntity.getAppId());
		jobDataMap.put("scheduleId", scheduleEntity.getScheduleId());
		jobDataMap.put("scalingAction", scalingAction);

		// The minimum and maximum instance count need to be set when the
		// scaling action has to be started.
		if (scalingAction.equals(ScalingActionEnum.START.getAction())) {
			jobDataMap.put("instanceMinCount", scheduleEntity.getInstanceMinCount());
			jobDataMap.put("instanceMaxCount", scheduleEntity.getInstanceMaxCount());
		}
	}
}
