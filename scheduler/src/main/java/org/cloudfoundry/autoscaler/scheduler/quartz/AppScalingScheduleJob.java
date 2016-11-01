package org.cloudfoundry.autoscaler.scheduler.quartz;

import java.io.IOException;
import java.net.HttpURLConnection;
import java.net.URL;

import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.Logger;
import org.cloudfoundry.autoscaler.scheduler.dao.ActiveScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.ActiveScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.JobActionEnum;
import org.cloudfoundry.autoscaler.scheduler.util.ScalingEngineUtil;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.scheduling.quartz.QuartzJobBean;
import org.springframework.stereotype.Component;

/**
 * QuartzJobBean class that executes the job
 */
@Component
abstract class AppScalingScheduleJob extends QuartzJobBean {
	private Logger logger = LogManager.getLogger(this.getClass());

	@Value("${autoscaler.scalingengine.url}")
	private String scalingEngineUrl;

	@Value("${scalingenginejob.refire.interval}")
	long jobRefireInterval;

	@Value("${scalingenginejob.refire.maxcount}")
	int maxJobRefireCount;

	@Autowired
	private ScalingEngineUtil scalingEngineUtil;

	@Autowired
	ActiveScheduleDao activeScheduleDao;

	@Autowired
	MessageBundleResourceHelper messageBundleResourceHelper;

	void notifyScalingEngine(ActiveScheduleEntity activeScheduleEntity, JobActionEnum scalingAction) {
		String appId = activeScheduleEntity.getAppId();
		Long scheduleId = activeScheduleEntity.getId();

		try {
			URL url = new URL(scalingEngineUrl + "/v1/apps/" + appId + "/active_schedule/" + scheduleId);

			HttpURLConnection scalingEngineConnection = scalingEngineUtil.getConnection(url, activeScheduleEntity);
			handleResponse(activeScheduleEntity, scalingAction, scalingEngineConnection);

		} catch (IOException e) {
			String message = messageBundleResourceHelper.lookupMessage("scalingengine.notification.error",
					e.getMessage(), appId, scheduleId, scalingAction);
			logger.error(message);
		} finally {
			scalingEngineUtil.close();
		}

	}

	private void handleResponse(ActiveScheduleEntity activeScheduleEntity, JobActionEnum scalingAction,
			HttpURLConnection scalingEngineConnection) throws IOException {
		int responseCode = scalingEngineConnection.getResponseCode();
		String responseMessage = scalingEngineConnection.getResponseMessage();
		String appId = activeScheduleEntity.getAppId();
		Long scheduleId = activeScheduleEntity.getId();
		if (responseCode >= 400 && responseCode < 500) {
			String message = messageBundleResourceHelper.lookupMessage("scalingengine.notification.client.error",
					responseCode, responseMessage, appId, scheduleId, scalingAction);
			logger.error(message);
		} else if (responseCode >= 500) {
			String message = messageBundleResourceHelper.lookupMessage("scalingengine.notification.failed",
					responseCode, responseMessage, appId, scheduleId, scalingAction);
			logger.error(message);
		} else {
			String message = messageBundleResourceHelper.lookupMessage("scalingengine.notification.success", appId,
					scheduleId, scalingAction);
			logger.info(message);

		}
	}
}
