package org.cloudfoundry.autoscaler.scheduler.quartz;

import java.time.LocalDateTime;
import java.time.ZonedDateTime;
import java.util.TimeZone;
import org.cloudfoundry.autoscaler.scheduler.util.DateHelper;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper;
import org.quartz.JobDataMap;
import org.quartz.JobExecutionContext;
import org.quartz.JobExecutionException;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

@Component
public class AppScalingSpecificDateScheduleStartJob extends AppScalingScheduleStartJob {
  private Logger logger = LoggerFactory.getLogger(this.getClass());

  @Override
  boolean shouldExecuteStartJob(
      JobExecutionContext jobExecutionContext,
      ZonedDateTime startJobStartTime,
      ZonedDateTime endJobStartTime)
      throws JobExecutionException {

    if (super.shouldExecuteStartJob(jobExecutionContext, startJobStartTime, endJobStartTime)) {

      boolean isStartTimeBeforeEnd = startJobStartTime.isBefore(endJobStartTime);
      if (!isStartTimeBeforeEnd) {
        JobDataMap jobDataMap = jobExecutionContext.getJobDetail().getJobDataMap();
        String message =
            messageBundleResourceHelper.lookupMessage(
                "scheduler.job.start.specificdate.schedule.skipped",
                DateHelper.convertLocalDateTimeToString(
                    (LocalDateTime) jobDataMap.get(ScheduleJobHelper.END_JOB_START_TIME)),
                jobExecutionContext.getJobDetail().getKey(),
                jobDataMap.getString(ScheduleJobHelper.APP_ID),
                jobDataMap.getLong(ScheduleJobHelper.SCHEDULE_ID));
        logger.warn(message);
      }

      return isStartTimeBeforeEnd;
    }
    return false;
  }

  @Override
  ZonedDateTime calculateEndJobStartTime(JobExecutionContext jobExecutionContext) {
    String timeZone =
        jobExecutionContext.getJobDetail().getJobDataMap().getString(ScheduleJobHelper.TIMEZONE);
    LocalDateTime endDateTime =
        (LocalDateTime)
            jobExecutionContext
                .getJobDetail()
                .getJobDataMap()
                .get(ScheduleJobHelper.END_JOB_START_TIME);

    return DateHelper.getZonedDateTime(endDateTime, TimeZone.getTimeZone(timeZone));
  }
}
