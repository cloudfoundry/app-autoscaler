package org.cloudfoundry.autoscaler.scheduler.util;

import java.util.Date;

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
		JobKey jobKey = new JobKey(name, scheduleTypeEnum.getScheduleIdentifier());
		return jobKey;
	}
	
	public static JobDetail buildJob(JobKey jobKey, Class<? extends Job> classType) {

		JobBuilder jobBuilder = JobBuilder.newJob(classType).withIdentity(jobKey).storeDurably();
		return jobBuilder.build();
	}

	public static TriggerKey generateTriggerKey(Long id, JobActionEnum jobActionEnum,
			ScheduleTypeEnum scheduleTypeEnum) {
		String name = id + jobActionEnum.getJobIdSuffix();
		TriggerKey triggerKey = new TriggerKey(name, scheduleTypeEnum.getScheduleIdentifier());
		return triggerKey;
	}
	
	public static Trigger buildTrigger(TriggerKey triggerKey, JobKey jobKey, Date triggerDate) {

		TriggerBuilder<Trigger> trigger = TriggerBuilder.newTrigger().withIdentity(triggerKey);

		trigger.withSchedule(SimpleScheduleBuilder.simpleSchedule()).startAt(triggerDate);

		if (jobKey != null) {
			trigger.forJob(jobKey);
		}

		return trigger.build();
	}

}
