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
 * 
 *
 */
public class ScheduleJobHelper {

	private static JobKey buildJobKey(String jobId) {
		return new JobKey(jobId);
	}
	
	public static String generateJobKey(Long jobId, JobActionEnum jobActionEnum) {
		return jobId + jobActionEnum.getJobIdSuffix();
	}

	public static JobDetail buildJob(String id, Class<? extends Job> classType) {
		JobKey jobKey = buildJobKey(id);
		JobBuilder jobBuilder = JobBuilder.newJob(classType).withIdentity(jobKey).storeDurably();
		return jobBuilder.build();
	}

	private static TriggerKey buildTiggerKey(String triggerId) {
		return new TriggerKey(triggerId, triggerId + "-trigger");
	}

	public static Trigger buildTrigger(String id, JobKey jobKey, Date triggerDate) {

		TriggerKey triggerKey = buildTiggerKey(id);
		TriggerBuilder<Trigger> trigger = TriggerBuilder.newTrigger().withIdentity(triggerKey);

		trigger.withSchedule(SimpleScheduleBuilder.simpleSchedule()).startAt(triggerDate);

		if (jobKey != null) {
			trigger.forJob(jobKey);
		}

		return trigger.build();
	}

}
