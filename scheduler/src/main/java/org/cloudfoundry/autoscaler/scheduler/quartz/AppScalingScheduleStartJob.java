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
import org.quartz.Scheduler;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Transactional;

@Component
public class AppScalingScheduleStartJob extends AppScalingScheduleJob {
	private Logger logger = LogManager.getLogger(this.getClass());

	@Autowired
	Scheduler scheduler;

	@Override
	public void executeInternal(JobExecutionContext jobExecutionContext) throws JobExecutionException {
		JobActionEnum jobStart = JobActionEnum.START;

		JobDataMap jobDataMap = jobExecutionContext.getJobDetail().getJobDataMap();
		ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(jobDataMap);

		String executingMessage = messageBundleResourceHelper.lookupMessage("scheduler.job.start",
				jobExecutionContext.getJobDetail().getKey(), activeScheduleEntity.getAppId(),
				activeScheduleEntity.getId(), jobStart, activeScheduleEntity.getInstanceMinCount(),
				activeScheduleEntity.getInstanceMaxCount(), activeScheduleEntity.getInitialMinInstanceCount());
		logger.info(executingMessage);

		// Persist the active schedule
		saveActiveSchedule(activeScheduleEntity, jobExecutionContext);

		notifyScalingEngine(activeScheduleEntity, jobStart, jobExecutionContext);

	}

	@Transactional
	private void saveActiveSchedule(ActiveScheduleEntity activeScheduleEntity,
			JobExecutionContext jobExecutionContext) {
		JobDataMap jobDataMap = jobExecutionContext.getJobDetail().getJobDataMap();
		boolean activeScheduleTableTaskDone = jobDataMap.getBoolean(ScheduleJobHelper.ACTIVE_SCHEDULE_TABLE_TASK_DONE);
		if (!activeScheduleTableTaskDone) {

			try {
				activeScheduleDao.create(activeScheduleEntity);
				jobDataMap.put(ScheduleJobHelper.ACTIVE_SCHEDULE_TABLE_TASK_DONE, true);
			} catch (DatabaseValidationException dve) {

				String errorMessage = messageBundleResourceHelper.lookupMessage(
						"database.error.create.activeschedule.failed", dve.getMessage(),
						activeScheduleEntity.getAppId(), activeScheduleEntity.getId());
				logger.error(errorMessage, dve);

				//Reschedule Job
				handleJobRescheduling(jobExecutionContext, ScheduleJobHelper.RescheduleCount.ACTIVE_SCHEDULE,
						maxJobRescheduleCount);
			}
		}

	}
}
