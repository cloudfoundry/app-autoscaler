package org.cloudfoundry.autoscaler.scheduler.util;

import java.time.LocalTime;
import java.time.ZonedDateTime;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Date;
import java.util.List;
import java.util.TimeZone;
import org.cloudfoundry.autoscaler.scheduler.entity.ActiveScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.quartz.CronScheduleBuilder;
import org.quartz.Job;
import org.quartz.JobBuilder;
import org.quartz.JobDataMap;
import org.quartz.JobDetail;
import org.quartz.JobKey;
import org.quartz.SimpleScheduleBuilder;
import org.quartz.Trigger;
import org.quartz.TriggerBuilder;
import org.quartz.TriggerKey;

/** Helper class for scheduler */
public class ScheduleJobHelper {

  public enum RescheduleCount {
    ACTIVE_SCHEDULE,
    SCALING_ENGINE_NOTIFICATION
  }

  public static final String APP_ID = "appId";
  public static final String SCHEDULE_ID = "scheduleId";
  public static final String TIMEZONE = "timeZone";
  public static final String INITIAL_MIN_INSTANCE_COUNT = "initialMinInstanceCount";
  public static final String INSTANCE_MIN_COUNT = "instanceMinCount";
  public static final String INSTANCE_MAX_COUNT = "instanceMaxCount";
  public static final String DEFAULT_INSTANCE_MIN_COUNT = "defaultInstanceMinCount";
  public static final String DEFAULT_INSTANCE_MAX_COUNT = "defaultInstanceMaxCount";
  public static final String START_JOB_IDENTIFIER = "startJobIdentifier";
  public static final String END_JOB_START_TIME = "endJobStartTime";
  public static final String END_JOB_CRON_EXPRESSION = "endJobCronExpression";
  public static final String ACTIVE_SCHEDULE_TABLE_CREATE_TASK_DONE =
      "activeScheduleTableCreateTask";
  public static final String CREATE_END_JOB_TASK_DONE = "endJobScheduleTask";

  public static JobDetail buildJob(JobKey jobKey, Class<? extends Job> classType) {

    JobBuilder jobBuilder = JobBuilder.newJob(classType).withIdentity(jobKey).storeDurably();
    return jobBuilder.build();
  }

  public static Trigger buildTrigger(
      TriggerKey triggerKey, JobKey jobKey, ZonedDateTime triggerDate) {

    TriggerBuilder<Trigger> trigger = TriggerBuilder.newTrigger().withIdentity(triggerKey);

    trigger
        .withSchedule(
            SimpleScheduleBuilder.simpleSchedule().withMisfireHandlingInstructionFireNow())
        .startAt(Date.from(triggerDate.toInstant()));
    if (jobKey != null) {
      trigger.forJob(jobKey);
    }

    return trigger.build();
  }

  public static Trigger buildCronTrigger(
      TriggerKey triggerKey,
      JobKey jobKey,
      RecurringScheduleEntity scheduleEntity,
      LocalTime scheduleTime) {
    TriggerBuilder<Trigger> trigger = TriggerBuilder.newTrigger().withIdentity(triggerKey);
    TimeZone timeZone = TimeZone.getTimeZone(scheduleEntity.getTimeZone());

    trigger.withSchedule(
        CronScheduleBuilder.cronSchedule(
                convertRecurringScheduleToCronExpression(scheduleTime, scheduleEntity))
            .inTimeZone(timeZone)
            .withMisfireHandlingInstructionFireAndProceed());

    if (scheduleEntity.getStartDate() != null) {
      ZonedDateTime startAt =
          DateHelper.getZonedDateTime(
              scheduleEntity.getStartDate(), TimeZone.getTimeZone(scheduleEntity.getTimeZone()));
      trigger.startAt(Date.from(startAt.toInstant()));
    }

    if (scheduleEntity.getEndDate() != null) {
      // Adding a day because Quartz recognizes the end time when the trigger is no longer fire so
      // it should be the next
      // day of policy's endDate. For example, if in policy the end date is "16/01/2016", then the
      // trigger will no longer
      // fire from "17/01/2016 00:00"
      ZonedDateTime endAt =
          DateHelper.getZonedDateTime(
              scheduleEntity.getEndDate().plusDays(1),
              TimeZone.getTimeZone(scheduleEntity.getTimeZone()));
      trigger.endAt(Date.from(endAt.toInstant()));
    }

    if (jobKey != null) {
      trigger.forJob(jobKey);
    }

    return trigger.build();
  }

  public static String convertRecurringScheduleToCronExpression(
      LocalTime scheduleTime, RecurringScheduleEntity recurringScheduleEntity) {
    int min = scheduleTime.getMinute();
    int hour = scheduleTime.getHour();

    String dayOfWeek = convertArrayToDayOfWeekString(recurringScheduleEntity.getDaysOfWeek());
    String dayOfMonth = convertArrayToDayOfMonthString(recurringScheduleEntity.getDaysOfMonth());

    return String.format("00 %02d %02d %s * %s *", min, hour, dayOfMonth, dayOfWeek);
  }

  private static String convertArrayToDayOfWeekString(int[] dayOfWeek) {
    String cronExpression = "?";
    if (dayOfWeek != null) {
      cronExpression = String.join(",", convertDayOfWeekToQuartzDayOfWeek(dayOfWeek));
    }

    return cronExpression;
  }

  private static List<String> convertDayOfWeekToQuartzDayOfWeek(int[] dayOfWeek) {
    List<String> quartzDayOfWeek = new ArrayList<>();

    for (int day : dayOfWeek) {
      quartzDayOfWeek.add(DateHelper.convertIntToDayOfWeek(day));
    }
    return quartzDayOfWeek;
  }

  private static String convertArrayToDayOfMonthString(int[] dayOfMonth) {
    String cronExpression = "?";
    if (dayOfMonth != null) {
      cronExpression = Arrays.toString(dayOfMonth).replaceAll("[\\[\\]\\s]", "");
    }
    return cronExpression;
  }

  public static ActiveScheduleEntity setupActiveSchedule(JobDataMap jobDataMap) {

    ActiveScheduleEntity activeScheduleEntity = new ActiveScheduleEntity();

    activeScheduleEntity.setAppId(jobDataMap.getString(APP_ID));
    activeScheduleEntity.setId(jobDataMap.getLongValue(SCHEDULE_ID));
    activeScheduleEntity.setInstanceMinCount(jobDataMap.getIntValue(INSTANCE_MIN_COUNT));
    activeScheduleEntity.setInstanceMaxCount(jobDataMap.getIntValue(INSTANCE_MAX_COUNT));

    // Initial min instance count can be null
    activeScheduleEntity.setInitialMinInstanceCount(
        (Integer) jobDataMap.get(INITIAL_MIN_INSTANCE_COUNT));

    return activeScheduleEntity;
  }
}
