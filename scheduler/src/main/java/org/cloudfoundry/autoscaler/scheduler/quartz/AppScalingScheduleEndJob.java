package org.cloudfoundry.autoscaler.scheduler.quartz;

import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.Logger;
import org.cloudfoundry.autoscaler.scheduler.entity.ActiveScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.JobActionEnum;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.DatabaseValidationException;
import org.quartz.JobDataMap;
import org.quartz.JobExecutionContext;
import org.quartz.JobExecutionException;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Transactional;

@Component
public class AppScalingScheduleEndJob extends AppScalingScheduleJob {
	private Logger logger = LogManager.getLogger(this.getClass());

	@Override
	public void executeInternal(JobExecutionContext jobExecutionContext) throws JobExecutionException {
		JobActionEnum jobEnd = JobActionEnum.END;

		JobDataMap jobDataMap = jobExecutionContext.getJobDetail().getJobDataMap();
		ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);

		String executingMessage = messageBundleResourceHelper.lookupMessage("scheduler.job.start",
				jobExecutionContext.getJobDetail().getKey(), activeScheduleEntity.getAppId(),
				activeScheduleEntity.getId(), jobEnd, activeScheduleEntity.getInstanceMinCount(),
				activeScheduleEntity.getInstanceMaxCount(), activeScheduleEntity.getInitialMinInstanceCount());
		logger.info(executingMessage);

		// Delete the active schedule
		deleteActiveSchedule(activeScheduleEntity, jobExecutionContext);

		notifyScalingEngine(activeScheduleEntity, jobEnd, jobExecutionContext);

	}

	@Transactional
	private void deleteActiveSchedule(ActiveScheduleEntity activeScheduleEntity,
			JobExecutionContext jobExecutionContext) {
		JobDataMap jobDataMap = jobExecutionContext.getJobDetail().getJobDataMap();
		boolean activeScheduleTableTaskDone = jobDataMap.getBoolean(ScheduleJobHelper.ACTIVE_SCHEDULE_TABLE_TASK_DONE);
		if (!activeScheduleTableTaskDone) {
			try {
				activeScheduleDao.delete(activeScheduleEntity.getId());
				jobDataMap.put(ScheduleJobHelper.ACTIVE_SCHEDULE_TABLE_TASK_DONE, true);
			} catch (DatabaseValidationException dve) {
				String errorMessage = messageBundleResourceHelper.lookupMessage(
						"database.error.delete.activeschedule.failed", dve.getMessage(),
						activeScheduleEntity.getAppId(), activeScheduleEntity.getId());
				logger.error(errorMessage, dve);

				//Reschedule Job
				handleJobRescheduling(jobExecutionContext, ScheduleJobHelper.RescheduleCount.ACTIVE_SCHEDULE,
						maxJobRescheduleCount);
			}

		}
	}
}
