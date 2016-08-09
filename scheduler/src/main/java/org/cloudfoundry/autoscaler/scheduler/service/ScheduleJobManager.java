package org.cloudfoundry.autoscaler.scheduler.service;

import java.util.Date;
import java.util.TimeZone;

import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.quartz.AppScalingScheduleJob;
import org.cloudfoundry.autoscaler.scheduler.util.DateHelper;
import org.cloudfoundry.autoscaler.scheduler.util.JobActionEnum;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleTypeEnum;
import org.cloudfoundry.autoscaler.scheduler.util.error.ValidationErrorResult;
import org.quartz.JobDataMap;
import org.quartz.JobDetail;
import org.quartz.JobKey;
import org.quartz.Scheduler;
import org.quartz.SchedulerException;
import org.quartz.Trigger;
import org.quartz.TriggerKey;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

/**
 * Service class to persist the schedule entity in the database and create
 * scheduled job.
 * 
 * 
 *
 */
@Service
public class ScheduleJobManager {
	@Autowired
	Scheduler scheduler;
	@Autowired
	ValidationErrorResult validationErrorResult;

	/**
	 * Creates simple job for specific date schedule for the application scaling using helper 
	 * methods. Here in two jobs are required, First job to tell the scaling decision maker
	 * scaling action needs to initiated Second job to tell the scaling decision maker scaling
	 * action needs to be ended.
	 * 
	 * @param specificDateScheduleEntity
	 * @throws SchedulerException 
	 * @throws Exception
	 */
	public void createSimpleJob(SpecificDateScheduleEntity specificDateScheduleEntity) {

		Long scheduleId = specificDateScheduleEntity.getId();

		// Build the job
		JobKey startJobKey = ScheduleJobHelper.generateJobKey(scheduleId, JobActionEnum.START,
				ScheduleTypeEnum.SPECIFIC_DATE);
		JobKey endJobKey = ScheduleJobHelper.generateJobKey(scheduleId, JobActionEnum.END,
				ScheduleTypeEnum.SPECIFIC_DATE);

		JobDetail startJobDetail = ScheduleJobHelper.buildJob(startJobKey, AppScalingScheduleJob.class);
		JobDetail endJobDetail = ScheduleJobHelper.buildJob(endJobKey, AppScalingScheduleJob.class);

		// Set the data in JobDetail for informing the scaling decision maker that scaling job needs to be started
		setupScalingScheduleJobData(startJobDetail, specificDateScheduleEntity, JobActionEnum.START);
		// Set the data in JobDetail for informing the scaling decision maker that scaling job needs to be ended.
		setupScalingScheduleJobData(endJobDetail, specificDateScheduleEntity, JobActionEnum.END);

		// Build the trigger
		TimeZone policyTimeZone = TimeZone.getTimeZone(specificDateScheduleEntity.getTimeZone());

		Date triggerStartDateTime = DateHelper.getDateWithZoneOffset(specificDateScheduleEntity.getStartDateTime(),
				policyTimeZone);
		Date triggerEndDateTime = DateHelper.getDateWithZoneOffset(specificDateScheduleEntity.getEndDateTime(),
				policyTimeZone);

		TriggerKey startTriggerKey = ScheduleJobHelper.generateTriggerKey(scheduleId, JobActionEnum.START,
				ScheduleTypeEnum.SPECIFIC_DATE);
		TriggerKey endTriggerKey = ScheduleJobHelper.generateTriggerKey(scheduleId, JobActionEnum.END,
				ScheduleTypeEnum.SPECIFIC_DATE);
		Trigger jobStartTrigger = ScheduleJobHelper.buildTrigger(startTriggerKey, startJobKey, triggerStartDateTime);
		Trigger jobEndTrigger = ScheduleJobHelper.buildTrigger(endTriggerKey, endJobKey, triggerEndDateTime);

		// Schedule the job
		try {
			scheduler.scheduleJob(startJobDetail, jobStartTrigger);
			scheduler.scheduleJob(endJobDetail, jobEndTrigger);

		} catch (SchedulerException se) {

			validationErrorResult.addErrorForQuartzSchedulerException(se, "scheduler.error.create.failed",
					"app_id=" + specificDateScheduleEntity.getAppId());
		}

	}

	/**
	 * Sets the data in the JobDetail object
	 * 
	 * @param jobDetail
	 * @param scheduleEntity
	 * @param jobAction 
	 */
	private void setupScalingScheduleJobData(JobDetail jobDetail, ScheduleEntity scheduleEntity,
			JobActionEnum jobAction) {

		JobDataMap jobDataMap = jobDetail.getJobDataMap();
		jobDataMap.put("appId", scheduleEntity.getAppId());
		jobDataMap.put("scheduleId", scheduleEntity.getId());
		jobDataMap.put("scalingAction", jobAction);

		// The minimum and maximum instance count need to be set when the
		// scaling action has to be started.
		if (jobAction == JobActionEnum.START) {
			jobDataMap.put("instanceMinCount", scheduleEntity.getInstanceMinCount());
			jobDataMap.put("instanceMaxCount", scheduleEntity.getInstanceMaxCount());
		} else {
			jobDataMap.put("instanceMinCount", scheduleEntity.getDefaultInstanceMinCount());
			jobDataMap.put("instanceMaxCount", scheduleEntity.getDefaultInstanceMaxCount());
		}
	}

	public void deleteSimpleJob(SpecificDateScheduleEntity specificDateScheduleEntity) {
		Long scheduleId = specificDateScheduleEntity.getId();

		JobKey startJobKey = ScheduleJobHelper.generateJobKey(scheduleId, JobActionEnum.START,
				ScheduleTypeEnum.SPECIFIC_DATE);
		JobKey endJobKey = ScheduleJobHelper.generateJobKey(scheduleId, JobActionEnum.END,
				ScheduleTypeEnum.SPECIFIC_DATE);

		try {
			scheduler.deleteJob(startJobKey);
			scheduler.deleteJob(endJobKey);
		} catch (SchedulerException se) {

			validationErrorResult.addErrorForQuartzSchedulerException(se, "scheduler.error.delete.failed",
					"app_id=" + specificDateScheduleEntity.getAppId());
		}
	}
}
