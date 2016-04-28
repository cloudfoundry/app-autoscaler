package org.cloudfoundry.autoscaler.metric.poller;

import java.util.ArrayList;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;
import java.util.Map.Entry;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.ScheduledThreadPoolExecutor;
import java.util.concurrent.TimeUnit;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.bean.InstanceMetrics;
import org.cloudfoundry.autoscaler.bean.Metric;
import org.cloudfoundry.autoscaler.common.util.CloudFoundryManager;
import org.cloudfoundry.autoscaler.common.util.ConfigManager;
import org.cloudfoundry.autoscaler.constant.Constants;
import org.cloudfoundry.autoscaler.data.couchdb.document.AppInstanceMetrics;
import org.cloudfoundry.autoscaler.metric.bean.CFInstanceStats;
import org.cloudfoundry.autoscaler.metric.bean.CFInstanceStats.Usage;
import org.cloudfoundry.autoscaler.metric.bean.CloudAppInstance;
import org.cloudfoundry.autoscaler.metric.monitor.MonitorController;
import org.cloudfoundry.autoscaler.metric.monitor.NamedThreadFactory;
import org.cloudfoundry.autoscaler.util.MetricConfigManager;

@SuppressWarnings({ "rawtypes" })
public class CFAppStatsPoller implements Runnable {
	private static final Logger logger = Logger.getLogger(CFAppStatsPoller.class);

	// in seconds
	private static final int DEFAULT_INTERVAL = 10;
	private static final int MAX_RETRY = 30;
	private int interval = ConfigManager.getInt(Constants.POLLING_INTERVAL, DEFAULT_INTERVAL);
	private int delay = ConfigManager.getInt(Constants.POLLING_WAIT, 60);

	private volatile boolean cancelled = false;

	// scheduled executor to periodically run the poller
	private static int appStatsPollerThreadCount = ConfigManager.getInt(Constants.POLLER_THREAD_COUNT, 50);
	private static ScheduledThreadPoolExecutor executor = new ScheduledThreadPoolExecutor(appStatsPollerThreadCount,
			new NamedThreadFactory("CFAppStatsPoller-Thread"));

	private String appId;
	private ScheduledFuture taskHandler = null;
	private int retry = 0;

	public CFAppStatsPoller(String appId) {
		this.appId = appId;
	}

	public void start() {
		if (taskHandler == null)
			taskHandler = executor.scheduleWithFixedDelay(this, delay, interval, TimeUnit.SECONDS);
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
			CFPollerManager.getInstance().removePoller(appId);
			logger.error(String.format("Remove poller for appId %s once MAX_RETRY count is reached", appId));
			return;
		}

		List<CloudAppInstance> statsList = null;
		MonitorController controller = MonitorController.getInstance();

		try {

			// if the app is not in "appMetric" Map, the app might be stopped or just bounded. Get app info first.
			if (!controller.isActiveApp(appId)) {
				String state = null;
				try {
					state = CloudFoundryManager.getInstance().getAppStateByAppId(appId);
				} catch (Exception e) {
					String message = e.getMessage();
					logger.error(String.format("Failed to get the state for app %s with exception %s", appId, message));
					if (message != null && message.contains("404")) {
						retry++;
					}
					return;
				}
				// if app the stopped, then do nothing
				if (state.equalsIgnoreCase(Constants.CF_APPLICATION_STATE_STOPPED)) {
					logger.debug(String.format("The app %s is stopped.", appId));
					return;
				}
			}

			// update appType & appName if it is null (just bounded)
			String appType = controller.getAppType(appId);
			String appName = controller.getAppNameById(appId);
			if (appType == null || appName == null || appType.isEmpty()) {
				String[] appNameAndType = CloudFoundryManager.getInstance().getAppNameAndType(appId);
				controller.updateAppNameAndType(appId, appNameAndType[0], appNameAndType[1]);
				appName = appNameAndType[0];
				appType = appNameAndType[1];
			}

			try {
				// check whether the poller need to be launched.
				// if an app is bounded without staging correctly, the recognized app type might be wrong, then the
				// poller is launched by default.
				// once the app is staged correctly, the app type is recognized. If no poller defined, we should stop
				// the poller.
				if (MetricConfigManager.getInstance().getEnabledMetric(appType, Constants.METRIC_SOURCE_POLLER,
						appId) == null) {
					logger.debug(new StringBuilder().append(appId)
							.append(" : poller is removed since no required poller metric defined."));
					controller.removePoller(appId);
					return;
				}

				// if an active record exists in "appMetric" Map or the app is started, then fetch the stats.
				logger.debug(String.format("Polling stats for app %s at time %s", appId, System.currentTimeMillis()));
				statsList = getAppStats(appId);
				retry = 0; // stop to count error once success.

			} catch (Exception e) {
				// if get exception here, maybe the target app is just stopped or deleted.
				String state = null;
				// double check whether the app is stopped
				try {
					state = CloudFoundryManager.getInstance().getAppStateByAppId(appId);
				} catch (Exception e1) {
					String e1Message = e1.getMessage();
					logger.error(
							String.format("Failed to get the state for app %s with exception %s", appId, e1Message));
					if (e1Message != null && e1Message.contains("404")) {
						retry++;
					}
					return;
				}
				// if app the stopped, then remove it from "appMetrics" map.
				if (state.equalsIgnoreCase(Constants.CF_APPLICATION_STATE_STOPPED)) {
					logger.debug(String.format("Purge the record for STOPPED app %s ", appId));
					controller.purgeAppFromMap(appId);
					return;
				}

				// if the app is not stopped, then log the exceptions to understand why the app stats API fails.
				String message = e.getMessage();
				if (message != null && message.contains("404")) {
					logger.warn("Application " + appId + " is not available for now.", e);
				} else {
					logger.error("Error when polling app " + appId + "(it might be stopped): " + message, e);
				}

				return;
			}

			// processing the statsList to AppInstanceMetrics
			if (statsList == null) {
				logger.warn("No running instance for app " + appId);
			} else {

				long now = System.currentTimeMillis();
				AppInstanceMetrics pollerMetrics = new AppInstanceMetrics();
				pollerMetrics.setAppId(appId);
				pollerMetrics.setAppName(appName);
				pollerMetrics.setAppType(appType);
				pollerMetrics.setTimestamp(now);
				List<InstanceMetrics> instanceMetricsList = new ArrayList<InstanceMetrics>(statsList.size());
				for (CloudAppInstance stats : statsList) {
					// update the memory quota info
					pollerMetrics.setMemQuota(stats.getMemQuotaMB());

					InstanceMetrics instanceMetric = new InstanceMetrics();
					instanceMetric.setInstanceId(stats.getInstanceIndex()); // set instance id == instance index, as we
																			// can't get instance id from poller
					instanceMetric.setInstanceIndex(Integer.parseInt(stats.getInstanceIndex()));
					instanceMetric.setTimestamp(now);
					instanceMetric.setStored(false);

					Metric metricMem = new Metric();
					metricMem.setCategory("cf-stats");
					metricMem.setGroup("Memory");
					metricMem.setName("Memory");
					metricMem.setUnit("MB");
					metricMem.setTimestamp(stats.getTimestamp());
					metricMem.setValue(String.valueOf(stats.getMemMB()));

					Metric metricCpu = new Metric();
					metricCpu.setCategory("cf-stats");
					metricCpu.setGroup("CPU");
					metricCpu.setName("CPU");
					metricCpu.setUnit("%");
					metricCpu.setTimestamp(stats.getTimestamp());
					metricCpu.setValue(String.valueOf(stats.getCpuPerc()));

					List<Metric> metrics = new LinkedList<Metric>();
					metrics.add(metricMem);
					metrics.add(metricCpu);

					instanceMetric.setMetrics(metrics);
					instanceMetricsList.add(instanceMetric);
				}

				pollerMetrics.setInstanceMetrics(instanceMetricsList);
				controller.processAppInstanceMetrics(pollerMetrics, Constants.METRIC_SOURCE_POLLER);
			} // end of else

		} catch (Exception e) {
			logger.error("Error when polling app " + appId, e);
		}
	}

	private List<CloudAppInstance> getAppStats(String appId) throws Exception {
		logger.debug("Calling CF to get stats of app " + appId);
		Map<String, Map<String, Object>> stats = CloudFoundryManager.getInstance().getApplicationStatsByAppId(appId);
		if (stats == null) {
			return null;
		}

		long timestamp = System.currentTimeMillis();
		ArrayList<CloudAppInstance> resultList = new ArrayList<CloudAppInstance>();
		for (Entry<String, Map<String, Object>> e : stats.entrySet()) {
			CFInstanceStats cfStat = new CFInstanceStats(e.getKey(), e.getValue());
			double cpuPerc = 0;
			double memMB = 0;
			Usage instUsage = cfStat.getUsage();
			double memQuotaMB = cfStat.getMemQuota() / (1024.0 * 1024.0);

			if (instUsage != null) {
				cpuPerc = 100 * instUsage.getCpu();
				memMB = instUsage.getMem() / (1024.0 * 1024.0);
				timestamp = instUsage.getTime().getTime();
			}
			CloudAppInstance resultStats = new CloudAppInstance(cfStat.getId(), cfStat.getHost(), cfStat.getCores(),
					cpuPerc, memMB, memQuotaMB, timestamp);
			logger.debug(String.format("inst = %16s  cores = %2d  cpu = %6.1f %%  mem = %6.1f MB mem_quota = %6.1f MB",
					cfStat.getId(), cfStat.getCores(), cpuPerc, memMB, memQuotaMB));
			resultList.add(resultStats);

		}

		return resultList;
	}

	public static void shutdown() {
		try {
			executor.shutdownNow();
		} catch (Exception e) {
			logger.error(e);
		}
	}
}
