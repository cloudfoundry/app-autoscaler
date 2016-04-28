package org.cloudfoundry.autoscaler.metric.poller;

import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.ScheduledThreadPoolExecutor;
import java.util.concurrent.TimeUnit;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.common.util.CloudFoundryManager;
import org.cloudfoundry.autoscaler.common.util.ConfigManager;
import org.cloudfoundry.autoscaler.constant.Constants;
import org.cloudfoundry.autoscaler.data.couchdb.document.AppInstanceMetrics;
import org.cloudfoundry.autoscaler.metric.monitor.MonitorController;
import org.cloudfoundry.autoscaler.metric.monitor.NamedThreadFactory;

@SuppressWarnings({ "rawtypes" })
public class CFAppInfoPoller implements Runnable {
	private static final Logger logger = Logger.getLogger(CFAppInfoPoller.class);

	private static final int MAX_RETRY = 10;
	private volatile boolean cancelled = false;

	// scheduled executor to periodically run the poller
	private static int appInfoPollerThreadCount = ConfigManager.getInt(Constants.POLLER_THREAD_COUNT, 10);
	private static ScheduledThreadPoolExecutor executor = new ScheduledThreadPoolExecutor(appInfoPollerThreadCount,
			new NamedThreadFactory("CFAppInfoPoller-Thread"));

	private String appId;
	private ScheduledFuture taskHandler = null;
	private int retry = 0;

	public CFAppInfoPoller(String appId) {
		this.appId = appId;
	}

	public void start() {
		if (taskHandler == null)
			taskHandler = executor.scheduleWithFixedDelay(this, 0, 30, TimeUnit.SECONDS);
	}

	public void stop() {
		if (taskHandler != null && !taskHandler.isCancelled()) {
			taskHandler.cancel(true);
		}
		cancelled = true;
	}

	@Override
	public void run() {
		if (cancelled) {
			return;
		}

		if (retry >= MAX_RETRY) {
			CFPollerManager.getInstance().removeAppInfoPoller(appId);
			return;
		}

		String[] appInfo = null;
		try {
			appInfo = CloudFoundryManager.getInstance().getAppInfoByAppId(appId);
		} catch (Exception e) {
			logger.error("Error with polling app memory Quota " + appId + e.getMessage());
			retry++;
			return;
		}

		MonitorController controller = MonitorController.getInstance();
		String appType = controller.getAppType(appId);
		String appName = controller.getAppNameById(appId);
		if (appType == null || appName == null || appType.isEmpty()) {
			appName = appInfo[0];
			appType = appInfo[1];
			try {
				controller.updateAppNameAndType(appId, appName, appType);
			} catch (Exception e) {
				logger.error("Error when update AppName and AppType record for app " + appId, e);
			}
		}

		long now = System.currentTimeMillis();
		AppInstanceMetrics pollerMetrics = new AppInstanceMetrics();
		pollerMetrics.setAppId(appId);
		pollerMetrics.setAppName(appName);
		pollerMetrics.setAppType(appType);
		pollerMetrics.setTimestamp(now);
		pollerMetrics.setMemQuota(Double.parseDouble(appInfo[2]));
		pollerMetrics.setInstanceMetrics(null);

		controller.processAppInstanceMetrics(pollerMetrics, Constants.METRIC_SOURCE_POLLER);

		CFPollerManager.getInstance().removeAppInfoPoller(appId);
	}

	public static void shutdown() {
		try {
			executor.shutdownNow();
		} catch (Exception e) {
			logger.error(e);
		}
	}

}
