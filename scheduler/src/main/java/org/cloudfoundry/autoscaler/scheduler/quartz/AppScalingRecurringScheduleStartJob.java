package org.cloudfoundry.autoscaler.scheduler.quartz;

import java.text.ParseException;
import java.time.ZonedDateTime;
import java.util.TimeZone;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper;
import org.quartz.CronExpression;
import org.quartz.JobDataMap;
import org.quartz.JobExecutionContext;
import org.quartz.JobExecutionException;
import org.quartz.JobKey;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

@Component
public class AppScalingRecurringScheduleStartJob extends AppScalingScheduleStartJob {
  private Logger logger = LoggerFactory.getLogger(this.getClass());

  @Override
  ZonedDateTime calculateEndJobStartTime(JobExecutionContext jobExecutionContext)
      throws JobExecutionException {
    JobDataMap jobDataMap = jobExecutionContext.getJobDetail().getJobDataMap();
    String timeZone = jobDataMap.getString(ScheduleJobHelper.TIMEZONE);
    String expression = jobDataMap.getString(ScheduleJobHelper.END_JOB_CRON_EXPRESSION);

    CronExpression cronExpression;
    try {
      cronExpression = new CronExpression(expression);
      cronExpression.setTimeZone(TimeZone.getTimeZone(timeZone));
    } catch (ParseException pe) {
      JobKey jobKey = jobExecutionContext.getJobDetail().getKey();
      String appId = jobDataMap.getString(ScheduleJobHelper.APP_ID);
      Long scheduleId = jobDataMap.getLong(ScheduleJobHelper.SCHEDULE_ID);
      String errorMessage =
          messageBundleResourceHelper.lookupMessage(
              "scheduler.job.cronexpression.parse.failed",
              pe.getMessage(),
              expression,
              jobKey,
              appId,
              scheduleId);
      logger.error(errorMessage, pe);
      throw new JobExecutionException(errorMessage, pe);
    }
    // Gets the next fire time for the end job which is after the current start job's fire time.
    return ZonedDateTime.ofInstant(
        cronExpression.getNextValidTimeAfter(jobExecutionContext.getFireTime()).toInstant(),
        TimeZone.getTimeZone(timeZone).toZoneId());
  }
}
