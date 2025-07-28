package org.cloudfoundry.autoscaler.scheduler.service;

import java.time.LocalDateTime;
import java.time.LocalTime;
import java.time.ZonedDateTime;
import java.util.TimeZone;
import org.cloudfoundry.autoscaler.scheduler.dao.ActiveScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.ActiveScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.quartz.AppScalingRecurringScheduleStartJob;
import org.cloudfoundry.autoscaler.scheduler.quartz.AppScalingSpecificDateScheduleStartJob;
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

/** Service class to persist the schedule entity in the database and create scheduled job. */
@Service
class ScheduleJobManager {
  @Autowired private Scheduler scheduler;

  @Autowired private ActiveScheduleDao activeScheduleDao;

  @Autowired private ValidationErrorResult validationErrorResult;

  /**
   * Creates simple job for specific date schedule for the application scaling using helper methods.
   * Here in two jobs are required, First job to tell the scaling decision maker scaling action
   * needs to initiated Second job to tell the scaling decision maker scaling action needs to be
   * ended.
   */
  void createSimpleJob(SpecificDateScheduleEntity specificDateScheduleEntity) {

    Long scheduleId = specificDateScheduleEntity.getId();
    String keyName = scheduleId + JobActionEnum.START.getJobIdSuffix();

    // Build the job
    JobKey startJobKey =
        new JobKey(keyName, ScheduleTypeEnum.SPECIFIC_DATE.getScheduleIdentifier());

    JobDetail jobDetail =
        ScheduleJobHelper.buildJob(startJobKey, AppScalingSpecificDateScheduleStartJob.class);

    // Set the data in JobDetail for informing the scaling engine that scaling job needs to be
    // started
    setupCommonScalingData(jobDetail, specificDateScheduleEntity);
    setupSpecificDateScheduleScalingData(jobDetail, specificDateScheduleEntity.getEndDateTime());

    // Build the trigger
    TimeZone policyTimeZone = TimeZone.getTimeZone(specificDateScheduleEntity.getTimeZone());

    ZonedDateTime triggerStartDateTime =
        DateHelper.getZonedDateTime(specificDateScheduleEntity.getStartDateTime(), policyTimeZone);

    TriggerKey startTriggerKey =
        new TriggerKey(keyName, ScheduleTypeEnum.SPECIFIC_DATE.getScheduleIdentifier());
    Trigger jobStartTrigger =
        ScheduleJobHelper.buildTrigger(startTriggerKey, startJobKey, triggerStartDateTime);

    // Schedule the job
    try {
      scheduler.scheduleJob(jobDetail, jobStartTrigger);

    } catch (SchedulerException se) {

      validationErrorResult.addErrorForQuartzSchedulerException(
          se,
          "scheduler.error.create.failed",
          "app_id=" + specificDateScheduleEntity.getAppId(),
          se.getMessage());
    }
  }

  void createCronJob(RecurringScheduleEntity recurringScheduleEntity) {
    Long scheduleId = recurringScheduleEntity.getId();
    String keyName = scheduleId + JobActionEnum.START.getJobIdSuffix();

    JobKey startJobKey = new JobKey(keyName, ScheduleTypeEnum.RECURRING.getScheduleIdentifier());

    // Build the job
    JobDetail jobStartDetail =
        ScheduleJobHelper.buildJob(startJobKey, AppScalingRecurringScheduleStartJob.class);

    // Build the trigger
    LocalTime triggerStartTime = recurringScheduleEntity.getStartTime();

    // Set the data in JobDetail for informing the scaling engine that scaling job needs to be
    // started
    String cronExpression =
        ScheduleJobHelper.convertRecurringScheduleToCronExpression(
            recurringScheduleEntity.getEndTime(), recurringScheduleEntity);
    setupCommonScalingData(jobStartDetail, recurringScheduleEntity);
    setupRecurringScheduleScalingData(jobStartDetail, cronExpression);

    TriggerKey startTriggerKey =
        new TriggerKey(keyName, ScheduleTypeEnum.RECURRING.getScheduleIdentifier());

    Trigger jobStartTrigger =
        ScheduleJobHelper.buildCronTrigger(
            startTriggerKey, jobStartDetail.getKey(), recurringScheduleEntity, triggerStartTime);
    // Schedule the job
    try {
      scheduler.scheduleJob(jobStartDetail, jobStartTrigger);
    } catch (SchedulerException se) {

      validationErrorResult.addErrorForQuartzSchedulerException(
          se,
          "scheduler.error.create.failed",
          "app_id=" + recurringScheduleEntity.getAppId(),
          se.getMessage());
    }
  }

  /**
   * Sets the data in the JobDetail object
   *
   * @param jobDetail
   * @param scheduleEntity
   */
  private void setupCommonScalingData(JobDetail jobDetail, ScheduleEntity scheduleEntity) {
    JobDataMap jobDataMap = jobDetail.getJobDataMap();
    jobDataMap.put(ScheduleJobHelper.APP_ID, scheduleEntity.getAppId());
    jobDataMap.put(ScheduleJobHelper.SCHEDULE_ID, scheduleEntity.getId());
    jobDataMap.put(ScheduleJobHelper.TIMEZONE, scheduleEntity.getTimeZone());
    jobDataMap.put(ScheduleJobHelper.INSTANCE_MIN_COUNT, scheduleEntity.getInstanceMinCount());
    jobDataMap.put(ScheduleJobHelper.INSTANCE_MAX_COUNT, scheduleEntity.getInstanceMaxCount());
    jobDataMap.put(
        ScheduleJobHelper.INITIAL_MIN_INSTANCE_COUNT, scheduleEntity.getInitialMinInstanceCount());
    jobDataMap.put(
        ScheduleJobHelper.DEFAULT_INSTANCE_MIN_COUNT, scheduleEntity.getDefaultInstanceMinCount());
    jobDataMap.put(
        ScheduleJobHelper.DEFAULT_INSTANCE_MAX_COUNT, scheduleEntity.getDefaultInstanceMaxCount());

    jobDataMap.put(ScheduleJobHelper.RescheduleCount.ACTIVE_SCHEDULE.name(), 1);
    jobDataMap.put(ScheduleJobHelper.RescheduleCount.SCALING_ENGINE_NOTIFICATION.name(), 1);
    jobDataMap.put(ScheduleJobHelper.ACTIVE_SCHEDULE_TABLE_CREATE_TASK_DONE, false);
    jobDataMap.put(ScheduleJobHelper.CREATE_END_JOB_TASK_DONE, false);
  }

  private void setupSpecificDateScheduleScalingData(
      JobDetail jobDetail, LocalDateTime endJobStartTime) {
    JobDataMap jobDataMap = jobDetail.getJobDataMap();
    jobDataMap.put(ScheduleJobHelper.END_JOB_START_TIME, endJobStartTime);
  }

  private void setupRecurringScheduleScalingData(JobDetail jobDetail, String cronExpression) {
    JobDataMap jobDataMap = jobDetail.getJobDataMap();
    jobDataMap.put(ScheduleJobHelper.END_JOB_CRON_EXPRESSION, cronExpression);
  }

  void deleteJob(String appId, Long scheduleId, ScheduleTypeEnum scheduleTypeEnum) {
    deleteJobFromQuartz(
        appId,
        scheduleId + JobActionEnum.START.getJobIdSuffix(),
        scheduleTypeEnum.getScheduleIdentifier());

    ActiveScheduleEntity activeScheduleEntity = activeScheduleDao.find(scheduleId);
    if (activeScheduleEntity != null) {
      String jobName =
          scheduleId
              + JobActionEnum.END.getJobIdSuffix()
              + "_"
              + activeScheduleEntity.getStartJobIdentifier();
      deleteJobFromQuartz(appId, jobName, "Schedule");
    }
  }

  private void deleteJobFromQuartz(String appId, String jobName, String jobGroup) {
    JobKey jobKey = new JobKey(jobName, jobGroup);
    try {
      scheduler.deleteJob(jobKey);
    } catch (SchedulerException se) {
      validationErrorResult.addErrorForQuartzSchedulerException(
          se, "scheduler.error.delete.failed", "app_id=" + appId, se.getMessage());
    }
  }
}
