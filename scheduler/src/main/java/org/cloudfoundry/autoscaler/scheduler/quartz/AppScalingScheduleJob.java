package org.cloudfoundry.autoscaler.scheduler.quartz;

import java.time.ZonedDateTime;
import java.util.concurrent.TimeUnit;
import org.cloudfoundry.autoscaler.scheduler.dao.ActiveScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.ActiveScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.JobActionEnum;
import org.cloudfoundry.autoscaler.scheduler.util.ScalingEngineUtil;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleJobHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
import org.quartz.JobDataMap;
import org.quartz.JobExecutionContext;
import org.quartz.Scheduler;
import org.quartz.SchedulerException;
import org.quartz.Trigger;
import org.quartz.TriggerKey;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.HttpEntity;
import org.springframework.http.HttpStatus;
import org.springframework.http.HttpStatusCode;
import org.springframework.scheduling.quartz.QuartzJobBean;
import org.springframework.stereotype.Component;
import org.springframework.web.client.HttpStatusCodeException;
import org.springframework.web.client.ResourceAccessException;
import org.springframework.web.client.RestOperations;

/** QuartzJobBean class that executes the job */
@Component
public abstract class AppScalingScheduleJob extends QuartzJobBean {
  private Logger logger = LoggerFactory.getLogger(this.getClass());

  @Value("${autoscaler.scalingengine.url}")
  private String scalingEngineUrl;

  @Value("${scalingenginejob.reschedule.interval.millisecond}")
  private long jobRescheduleIntervalMilliSecond;

  @Value("${scalingenginejob.reschedule.maxcount}")
  int maxJobRescheduleCount;

  @Value("${scalingengine.notification.reschedule.maxcount}")
  private int maxScalingEngineNotificationRescheduleCount;

  @Autowired ActiveScheduleDao activeScheduleDao;

  @Autowired private RestOperations restOperations;

  @Autowired MessageBundleResourceHelper messageBundleResourceHelper;

  void notifyScalingEngine(
      ActiveScheduleEntity activeScheduleEntity,
      JobActionEnum scalingAction,
      JobExecutionContext jobExecutionContext) {
    String appId = activeScheduleEntity.getAppId();
    Long scheduleId = activeScheduleEntity.getId();
    HttpEntity<ActiveScheduleEntity> requestEntity = new HttpEntity<>(activeScheduleEntity);

    try {
      String scalingEnginePathActiveSchedule =
          ScalingEngineUtil.getScalingEngineActiveSchedulePath(scalingEngineUrl, appId, scheduleId);

      if (scalingAction == JobActionEnum.START) {
        String message =
            messageBundleResourceHelper.lookupMessage(
                "scalingengine.notification.activeschedule.start", appId, scheduleId);
        logger.info(message);
        restOperations.put(scalingEnginePathActiveSchedule, requestEntity);
      } else {
        String message =
            messageBundleResourceHelper.lookupMessage(
                "scalingengine.notification.activeschedule.remove", appId, scheduleId);
        logger.info(message);
        restOperations.delete(scalingEnginePathActiveSchedule);
      }
    } catch (HttpStatusCodeException hce) {
      handleResponse(activeScheduleEntity, scalingAction, hce);
    } catch (ResourceAccessException rae) {
      String message =
          messageBundleResourceHelper.lookupMessage(
              "scalingengine.notification.error",
              rae.getMessage(),
              appId,
              scheduleId,
              scalingAction);
      logger.error(message, rae);
      handleJobRescheduling(
          jobExecutionContext,
          ScheduleJobHelper.RescheduleCount.SCALING_ENGINE_NOTIFICATION,
          maxScalingEngineNotificationRescheduleCount);
    }
  }

  private void handleResponse(
      ActiveScheduleEntity activeScheduleEntity,
      JobActionEnum scalingAction,
      HttpStatusCodeException hsce) {
    String appId = activeScheduleEntity.getAppId();
    Long scheduleId = activeScheduleEntity.getId();
    HttpStatusCode errorResponseCode = hsce.getStatusCode();
    if (errorResponseCode == HttpStatus.NOT_FOUND) {
      String message =
          messageBundleResourceHelper.lookupMessage(
              "scalingengine.notification.activeschedule.notFound", appId, scheduleId);
      logger.info(message, hsce);
    } else {
      String message =
          messageBundleResourceHelper.lookupMessage(
              "scalingengine.notification.failed",
              errorResponseCode.value(),
              hsce.getResponseBodyAsString(),
              appId,
              scheduleId,
              scalingAction);
      logger.error(message, hsce);
    }
  }

  void handleJobRescheduling(
      JobExecutionContext jobExecutionContext,
      ScheduleJobHelper.RescheduleCount retryCounter,
      int maxCount) {
    JobDataMap jobDataMap = jobExecutionContext.getJobDetail().getJobDataMap();
    String retryCounterTask = retryCounter.name(); // ACTIVE_SCHEDULE, SCALING_ENGINE_NOTIFICATION
    int jobFireCount = jobDataMap.getInt(retryCounterTask);
    String appId = jobDataMap.getString(ScheduleJobHelper.APP_ID);
    Long scheduleId = jobDataMap.getLong(ScheduleJobHelper.SCHEDULE_ID);
    TriggerKey triggerKey = jobExecutionContext.getTrigger().getKey();

    logger.info(
        "Rescheduling job for Trigger Key: "
            + triggerKey
            + ", Application Id: "
            + appId
            + ", Schedule Id: "
            + scheduleId);

    if (jobFireCount < maxCount) {
      ZonedDateTime newTriggerTime =
          ZonedDateTime.now()
              .plusNanos(TimeUnit.MILLISECONDS.toNanos(jobRescheduleIntervalMilliSecond));
      Trigger newTrigger = ScheduleJobHelper.buildTrigger(triggerKey, null, newTriggerTime);

      try {
        Scheduler scheduler = jobExecutionContext.getScheduler();
        jobDataMap.put(retryCounterTask, ++jobFireCount);
        scheduler.addJob(jobExecutionContext.getJobDetail(), true);
        scheduler.rescheduleJob(triggerKey, newTrigger);
      } catch (SchedulerException se) {
        String errorMessage =
            messageBundleResourceHelper.lookupMessage(
                "scheduler.job.reschedule.failed",
                se.getMessage(),
                triggerKey,
                appId,
                scheduleId,
                jobFireCount - 1);
        logger.error(errorMessage, se);
      }
    } else {
      String errorMessage =
          messageBundleResourceHelper.lookupMessage(
              "scheduler.job.reschedule.failed.max.reached",
              triggerKey,
              appId,
              scheduleId,
              maxCount,
              retryCounterTask);
      logger.error(errorMessage);
    }
  }
}
