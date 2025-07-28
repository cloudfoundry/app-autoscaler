package org.cloudfoundry.autoscaler.scheduler.quartz;

import org.cloudfoundry.autoscaler.scheduler.entity.ActiveScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.JobActionEnum;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.DatabaseValidationException;
import org.quartz.JobDataMap;
import org.quartz.JobExecutionContext;
import org.quartz.JobExecutionException;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Transactional;

@Component
public class AppScalingScheduleEndJob extends AppScalingScheduleJob {
  private Logger logger = LoggerFactory.getLogger(this.getClass());

  @Override
  public void executeInternal(JobExecutionContext jobExecutionContext)
      throws JobExecutionException {
    JobActionEnum jobEnd = JobActionEnum.END;

    JobDataMap jobDataMap = jobExecutionContext.getJobDetail().getJobDataMap();

    String appId = jobDataMap.getString(ScheduleJobHelper.APP_ID);
    long scheduleId = jobDataMap.getLong(ScheduleJobHelper.SCHEDULE_ID);

    String executingMessage =
        messageBundleResourceHelper.lookupMessage(
            "scheduler.job.start",
            jobExecutionContext.getJobDetail().getKey(),
            appId,
            scheduleId,
            jobEnd);
    logger.info(executingMessage);

    deleteActiveSchedule(jobExecutionContext);

    ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);
    notifyScalingEngine(activeScheduleEntity, jobEnd, jobExecutionContext);
  }

  @Transactional
  private void deleteActiveSchedule(JobExecutionContext jobExecutionContext)
      throws JobExecutionException {
    JobDataMap jobDataMap = jobExecutionContext.getJobDetail().getJobDataMap();
    String appId = jobDataMap.getString(ScheduleJobHelper.APP_ID);
    long scheduleId = jobDataMap.getLong(ScheduleJobHelper.SCHEDULE_ID);
    long startJobIdentifier = jobDataMap.getLong(ScheduleJobHelper.START_JOB_IDENTIFIER);

    try {
      activeScheduleDao.delete(scheduleId, startJobIdentifier);
    } catch (DatabaseValidationException dve) {
      String errorMessage =
          messageBundleResourceHelper.lookupMessage(
              "database.error.delete.activeschedule.failed", dve.getMessage(), appId, scheduleId);
      logger.error(errorMessage, dve);

      // Reschedule Job
      handleJobRescheduling(
          jobExecutionContext,
          ScheduleJobHelper.RescheduleCount.ACTIVE_SCHEDULE,
          maxJobRescheduleCount);

      throw new JobExecutionException(errorMessage, dve);
    }
  }
}
