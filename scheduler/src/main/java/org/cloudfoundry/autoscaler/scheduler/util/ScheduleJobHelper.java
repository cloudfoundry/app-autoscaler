package org.cloudfoundry.autoscaler.scheduler.util;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.Calendar;
import java.util.Date;
import java.util.List;
import java.util.TimeZone;

import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.quartz.CronScheduleBuilder;
import org.quartz.Job;
import org.quartz.JobBuilder;
import org.quartz.JobDetail;
import org.quartz.JobKey;
import org.quartz.SimpleScheduleBuilder;
import org.quartz.Trigger;
import org.quartz.TriggerBuilder;
import org.quartz.TriggerKey;

/**
 * Helper class for scheduler
 *
 */
public class ScheduleJobHelper {

	public static JobKey generateJobKey(Long id, JobActionEnum jobActionEnum, ScheduleTypeEnum scheduleTypeEnum) {
		String name = id + jobActionEnum.getJobIdSuffix();
		return new JobKey(name, scheduleTypeEnum.getScheduleIdentifier());
	}

	public static JobDetail buildJob(JobKey jobKey, Class<? extends Job> classType) {

		JobBuilder jobBuilder = JobBuilder.newJob(classType).withIdentity(jobKey).storeDurably();
		return jobBuilder.build();
	}

	public static TriggerKey generateTriggerKey(Long id, JobActionEnum jobActionEnum,
			ScheduleTypeEnum scheduleTypeEnum) {
		String name = id + jobActionEnum.getJobIdSuffix();
		return new TriggerKey(name, scheduleTypeEnum.getScheduleIdentifier());
	}

	public static Trigger buildTrigger(TriggerKey triggerKey, JobKey jobKey, Date triggerDate) {

		TriggerBuilder<Trigger> trigger = TriggerBuilder.newTrigger().withIdentity(triggerKey);

		trigger.withSchedule(SimpleScheduleBuilder.simpleSchedule()).startAt(triggerDate);

		if (jobKey != null) {
			trigger.forJob(jobKey);
		}

		return trigger.build();
	}

	public static Trigger buildCronTrigger(TriggerKey triggerKey, JobKey jobKey, RecurringScheduleEntity scheduleEntity,
			Date scheduleTime) {
		TriggerBuilder<Trigger> trigger = TriggerBuilder.newTrigger().withIdentity(triggerKey);
		TimeZone timeZone = TimeZone.getTimeZone(scheduleEntity.getTimeZone());

		trigger.withSchedule(
				CronScheduleBuilder.cronSchedule(convertRecurringScheduleToCronExpression(scheduleTime, scheduleEntity))
						.inTimeZone(timeZone));

		if (scheduleEntity.getStartDate() != null) {
			trigger.startAt(scheduleEntity.getStartDate());
		}

		if (scheduleEntity.getEndDate() != null) {
			trigger.endAt(scheduleEntity.getEndDate());
		}

		if (jobKey != null) {
			trigger.forJob(jobKey);
		}

		return trigger.build();
	}

	private static String convertRecurringScheduleToCronExpression(Date scheduleTime,
			RecurringScheduleEntity recurringScheduleEntity) {
		Calendar calendar = Calendar.getInstance();
		calendar.setTime(scheduleTime);

		int min = calendar.get(Calendar.MINUTE);
		int hour = calendar.get(Calendar.HOUR_OF_DAY);

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

}
