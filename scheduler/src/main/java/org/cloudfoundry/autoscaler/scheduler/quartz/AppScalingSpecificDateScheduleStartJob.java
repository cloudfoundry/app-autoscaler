package org.cloudfoundry.autoscaler.scheduler.quartz;

import java.util.Date;
import java.util.TimeZone;

import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.Logger;
import org.cloudfoundry.autoscaler.scheduler.util.DateHelper;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper;
import org.quartz.JobDataMap;
import org.quartz.JobExecutionContext;
import org.springframework.stereotype.Component;

@Component
public class AppScalingSpecificDateScheduleStartJob extends AppScalingScheduleStartJob {
	private Logger logger = LogManager.getLogger(this.getClass());

	@Override
	boolean shouldExecuteStartJob(JobExecutionContext jobExecutionContext, Date startJobStartTime,
			Date endJobStartTime) {
		boolean isStartTimeBeforeEnd = startJobStartTime.before(endJobStartTime);

		if (!isStartTimeBeforeEnd) {
			JobDataMap jobDataMap = jobExecutionContext.getJobDetail().getJobDataMap();
			String message = messageBundleResourceHelper.lookupMessage(
					"scheduler.job.start.specificdate.schedule.skipped",
					new Date(jobDataMap.getLong(ScheduleJobHelper.END_JOB_START_TIME)),
					jobExecutionContext.getJobDetail().getKey(), jobDataMap.getString(ScheduleJobHelper.APP_ID),
					jobDataMap.getLong(ScheduleJobHelper.SCHEDULE_ID));
			logger.warn(message);
		}

		return isStartTimeBeforeEnd;
	}

	@Override
	Date calculateEndJobStartTime(JobExecutionContext jobExecutionContext) {
		String timeZone = jobExecutionContext.getJobDetail().getJobDataMap().getString(ScheduleJobHelper.TIMEZONE);
		long endDateTime = jobExecutionContext.getJobDetail().getJobDataMap()
				.getLong(ScheduleJobHelper.END_JOB_START_TIME);

		return DateHelper.getDateWithZoneOffset(new Date(endDateTime), TimeZone.getTimeZone(timeZone));
	}

}
