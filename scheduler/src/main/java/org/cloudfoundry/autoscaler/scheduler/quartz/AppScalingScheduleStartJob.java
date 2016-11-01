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
public class AppScalingScheduleStartJob extends AppScalingScheduleJob {
	private Logger logger = LogManager.getLogger(this.getClass());

	@Override
	public void executeInternal(JobExecutionContext context) throws JobExecutionException {
		JobActionEnum jobStart = JobActionEnum.START;


		JobDataMap dataMap = context.getJobDetail().getJobDataMap();
		ActiveScheduleEntity activeScheduleEntity = ScheduleJobHelper.setupActiveSchedule(dataMap);

		String executingMessage = messageBundleResourceHelper.lookupMessage("scheduler.job.start",
				context.getJobDetail().getKey(), activeScheduleEntity.getAppId(), activeScheduleEntity.getId(),
				jobStart, activeScheduleEntity.getInstanceMinCount(), activeScheduleEntity.getInstanceMaxCount(),
				activeScheduleEntity.getInitialMinInstanceCount());
		logger.info(executingMessage);

		// Persist the active schedule
		saveActiveSchedule(activeScheduleEntity, context);

		notifyScalingEngine(activeScheduleEntity, jobStart);

	}

	@Transactional
	private void saveActiveSchedule(ActiveScheduleEntity activeScheduleEntity, JobExecutionContext jobExecutionContext) throws JobExecutionException {
		try {
			activeScheduleDao.create(activeScheduleEntity);
		} catch (DatabaseValidationException dve) {
			// Refire the job immediately
			String errorMessage = messageBundleResourceHelper
					.lookupMessage("database.error.create.activeschedule.failed", dve.getMessage());
			logger.error(errorMessage, dve);
			if(jobExecutionContext.getRefireCount() < maxJobRefireCount) {
				try {
					Thread.sleep(jobRefireInterval);
				}  catch (InterruptedException ie) {
					logger.error(ie.getMessage(), ie);
				}
				throw new JobExecutionException(dve, true);
			}
		}

	}
}
