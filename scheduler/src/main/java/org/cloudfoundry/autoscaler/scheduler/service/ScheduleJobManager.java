package org.cloudfoundry.autoscaler.scheduler.service;

import java.util.Date;
import java.util.TimeZone;

import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.quartz.AppScalingScheduleEndJob;
import org.cloudfoundry.autoscaler.scheduler.quartz.AppScalingScheduleStartJob;
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
class ScheduleJobManager {
	@Autowired
	private Scheduler scheduler;
	@Autowired
	private ValidationErrorResult validationErrorResult;

	/**
	 * Creates simple job for specific date schedule for the application scaling using helper 
	 * methods. Here in two jobs are required, First job to tell the scaling decision maker
	 * scaling action needs to initiated Second job to tell the scaling decision maker scaling
	 * action needs to be ended.
	 */
	void createSimpleJob(SpecificDateScheduleEntity specificDateScheduleEntity) {

		Long scheduleId = specificDateScheduleEntity.getId();

		// Build the job
		JobKey startJobKey = ScheduleJobHelper.generateJobKey(scheduleId, JobActionEnum.START,
				ScheduleTypeEnum.SPECIFIC_DATE);
		JobKey endJobKey = ScheduleJobHelper.generateJobKey(scheduleId, JobActionEnum.END,
				ScheduleTypeEnum.SPECIFIC_DATE);

		JobDetail startJobDetail = ScheduleJobHelper.buildJob(startJobKey, AppScalingScheduleStartJob.class);
		JobDetail endJobDetail = ScheduleJobHelper.buildJob(endJobKey, AppScalingScheduleEndJob.class);

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
					"app_id=" + specificDateScheduleEntity.getAppId(), se.getMessage());
		}

	}

	void createCronJob(RecurringScheduleEntity recurringScheduleEntity) {
		Long scheduleId = recurringScheduleEntity.getId();

		JobKey startJobKey = ScheduleJobHelper.generateJobKey(scheduleId, JobActionEnum.START,
				ScheduleTypeEnum.RECURRING);
		JobKey endJobKey = ScheduleJobHelper.generateJobKey(scheduleId, JobActionEnum.END, ScheduleTypeEnum.RECURRING);

		// Build the job
		JobDetail jobStartDetail = ScheduleJobHelper.buildJob(startJobKey, AppScalingScheduleStartJob.class);
		JobDetail jobEndDetail = ScheduleJobHelper.buildJob(endJobKey, AppScalingScheduleEndJob.class);

		// Set the data in JobDetail for informing the scaling decision maker that scaling job needs to be started
		setupScalingScheduleJobData(jobStartDetail, recurringScheduleEntity, JobActionEnum.START);
		// Set the data in JobDetail for informing the scaling decision maker that scaling job needs to be ended.
		setupScalingScheduleJobData(jobEndDetail, recurringScheduleEntity, JobActionEnum.END);

		// Build the trigger
		Date triggerStartTime = recurringScheduleEntity.getStartTime();
		Date triggerEndTime = recurringScheduleEntity.getEndTime();

		TriggerKey startTriggerKey = ScheduleJobHelper.generateTriggerKey(scheduleId, JobActionEnum.START,
				ScheduleTypeEnum.RECURRING);
		TriggerKey endTriggerKey = ScheduleJobHelper.generateTriggerKey(scheduleId, JobActionEnum.END,
				ScheduleTypeEnum.RECURRING);

		Trigger jobStartTrigger = ScheduleJobHelper.buildCronTrigger(startTriggerKey, jobStartDetail.getKey(),
				recurringScheduleEntity, triggerStartTime);
		Trigger jobEndTrigger = ScheduleJobHelper.buildCronTrigger(endTriggerKey, jobEndDetail.getKey(),
				recurringScheduleEntity, triggerEndTime);

		// Schedule the job
		try {
			scheduler.scheduleJob(jobStartDetail, jobStartTrigger);
			scheduler.scheduleJob(jobEndDetail, jobEndTrigger);

		} catch (SchedulerException se) {

			validationErrorResult.addErrorForQuartzSchedulerException(se, "scheduler.error.create.failed",
					"app_id=" + recurringScheduleEntity.getAppId(), se.getMessage());
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
		jobDataMap.put(ScheduleJobHelper.APP_ID, scheduleEntity.getAppId());
		jobDataMap.put(ScheduleJobHelper.SCHEDULE_ID, scheduleEntity.getId());
		jobDataMap.put(ScheduleJobHelper.SCALING_ACTION, jobAction.getStatus());

		// The minimum and maximum instance count need to be set when the
		// scaling action has to be started.
		if (jobAction == JobActionEnum.START) {
			jobDataMap.put(ScheduleJobHelper.INSTANCE_MIN_COUNT, scheduleEntity.getInstanceMinCount());
			jobDataMap.put(ScheduleJobHelper.INSTANCE_MAX_COUNT, scheduleEntity.getInstanceMaxCount());
			jobDataMap.put(ScheduleJobHelper.INITIAL_MIN_INSTANCE_COUNT, scheduleEntity.getInitialMinInstanceCount());
		} else {
			jobDataMap.put(ScheduleJobHelper.INSTANCE_MIN_COUNT, scheduleEntity.getDefaultInstanceMinCount());
			jobDataMap.put(ScheduleJobHelper.INSTANCE_MAX_COUNT, scheduleEntity.getDefaultInstanceMaxCount());
		}
	}

	void deleteJob(String appId, Long scheduleId, ScheduleTypeEnum scheduleTypeEnum) {

		JobKey startJobKey = ScheduleJobHelper.generateJobKey(scheduleId, JobActionEnum.START, scheduleTypeEnum);
		JobKey endJobKey = ScheduleJobHelper.generateJobKey(scheduleId, JobActionEnum.END, scheduleTypeEnum);

		try {
			scheduler.deleteJob(startJobKey);
			scheduler.deleteJob(endJobKey);
		} catch (SchedulerException se) {

			validationErrorResult.addErrorForQuartzSchedulerException(se, "scheduler.error.delete.failed",
					"app_id=" + appId, se.getMessage());
		}
	}
}
