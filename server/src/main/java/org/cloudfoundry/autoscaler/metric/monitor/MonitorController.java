package org.cloudfoundry.autoscaler.metric.monitor;

import java.io.IOException;
import java.util.Collection;
import java.util.HashMap;
import java.util.HashSet;
import java.util.Iterator;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.LinkedBlockingQueue;
import java.util.concurrent.ScheduledThreadPoolExecutor;
import java.util.concurrent.ThreadPoolExecutor;
import java.util.concurrent.TimeUnit;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.bean.InstanceMetrics;
import org.cloudfoundry.autoscaler.bean.Metric;
import org.cloudfoundry.autoscaler.bean.MonitorTriggerEvent;
import org.cloudfoundry.autoscaler.bean.Trigger;
import org.cloudfoundry.autoscaler.common.util.CloudFoundryManager;
import org.cloudfoundry.autoscaler.common.util.ConfigManager;
import org.cloudfoundry.autoscaler.constant.Constants;
import org.cloudfoundry.autoscaler.data.AutoScalingDataStore;
import org.cloudfoundry.autoscaler.data.couchdb.AutoScalingDataStoreFactory;
import org.cloudfoundry.autoscaler.data.couchdb.document.AppInstanceMetrics;
import org.cloudfoundry.autoscaler.data.couchdb.document.BoundApp;
import org.cloudfoundry.autoscaler.exceptions.TriggerNotFoundException;
import org.cloudfoundry.autoscaler.manager.ScalingEventManager;
import org.cloudfoundry.autoscaler.metric.bean.ApplicationMetrics;
import org.cloudfoundry.autoscaler.metric.poller.CFPollerManager;
import org.cloudfoundry.autoscaler.util.MetricConfigManager;

/**
 * 
 * peterw: this is a rename and rewrite of the original RunTime class in the
 * monitor. This class is the central controller that manages all monitoring
 * events and actions.
 */
public class MonitorController implements Runnable {

	private static final Logger logger = Logger.getLogger(MonitorController.class);
	private static final Logger loggerEvent = Logger.getLogger("triggerevent");
	private static final int DEFAULT_PURGE_TIME = 60;

	private static MonitorController instance = new MonitorController();
	private ConcurrentMap<String, StateMonitor> monitorMap;

	private ConcurrentMap<String, HashMap<String, Metric>> testMetricsMap = new ConcurrentHashMap<String, HashMap<String, Metric>>();

	// <appId, ApplicationMetrics>
	private ConcurrentMap<String, ApplicationMetrics> appMetricsMap = new ConcurrentHashMap<String, ApplicationMetrics>();

	// <serviceId, Map<appId, BoundApp>>
	private ConcurrentMap<String, Map<String, BoundApp>> serviceBoundAppsMap = new ConcurrentHashMap<String, Map<String, BoundApp>>();
	private ConcurrentMap<String, BoundApp> boundAppMap = new ConcurrentHashMap<String, BoundApp>();

	private BlockingQueue<AppInstanceMetrics> metricsQueue = new LinkedBlockingQueue<AppInstanceMetrics>();

	private ExecutorService processingExecutor = new ThreadPoolExecutor(1, 1, 0L, TimeUnit.MILLISECONDS,
			new LinkedBlockingQueue<Runnable>(), new NamedThreadFactory("processingExecutor"));
	private int threadCount = (int) (Runtime.getRuntime().availableProcessors() * 2);
	private ExecutorService metricsProcessingExecutor = new ThreadPoolExecutor(threadCount, threadCount, 0L,
			TimeUnit.MILLISECONDS, new LinkedBlockingQueue<Runnable>(),
			new NamedThreadFactory("metricsProcessingExecutor"));
	private ExecutorService scaleProcessExecutor = new ThreadPoolExecutor(threadCount, threadCount, 0L,
			TimeUnit.MILLISECONDS, new LinkedBlockingQueue<Runnable>(), new NamedThreadFactory("scaleProcessExecutor"));

	private ScheduledThreadPoolExecutor purgeAppMetricsMapExecutor = new ScheduledThreadPoolExecutor(1);
	private ScheduledThreadPoolExecutor purgeLocalCacheExecutor = new ScheduledThreadPoolExecutor(1);

	private volatile boolean processingStoped = false;

	private boolean store2db = true;

	private MonitorController() {
		monitorMap = new ConcurrentHashMap<String, StateMonitor>();
		processingExecutor.execute(this);

		purgeAppMetricsMapExecutor.setThreadFactory(new NamedThreadFactory("purgeAppMetricsMapExecutor"));
		purgeAppMetricsMapExecutor.scheduleWithFixedDelay(new PurgeAppMetricsMapThread(appMetricsMap),
				DEFAULT_PURGE_TIME, DEFAULT_PURGE_TIME, TimeUnit.SECONDS);

		purgeLocalCacheExecutor.setThreadFactory(new NamedThreadFactory("purgeLocalCacheExecutor"));
		purgeLocalCacheExecutor.scheduleWithFixedDelay(new PurgeLocalCacheThread(), 24, 24, TimeUnit.HOURS);

	}

	public static MonitorController getInstance() {
		return instance;
	}

	public boolean sotore2db() {
		return this.store2db;
	}

	public void addTrigger(Trigger t) throws Exception {
		String appId = t.getAppId();
		logger.info("add Trigger " + t.getMetric() + " for app: " + appId);
		AutoScalingDataStore storeService = AutoScalingDataStoreFactory.getAutoScalingDataStore();
		try {
			// store to db
			try {
				storeService.addTrigger(t);
			} catch (Exception e) {
				logger.warn(e.getMessage(), e);
			}

			// check if the corresponding state monitor exists, if not, create
			// one
			StateMonitor sm = this.getStateMonitor(appId);
			if (sm == null) {
				sm = this.createStateMonitor(appId);
			}
			sm.addTrigger(t);
		} catch (Exception e) {
			throw e;
		}
	}

	public void removeTrigger(String appId) throws TriggerNotFoundException {
		logger.info("remove Triggers for appId = " + appId);
		try {
			AutoScalingDataStore storeService = AutoScalingDataStoreFactory.getAutoScalingDataStore();
			storeService.removeTrigger(appId);
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		StateMonitor sm = this.getStateMonitor(appId);
		if (null == sm) {
			throw new TriggerNotFoundException("Trigger not found");
		}

		this.deleteStateMonitor(appId);

	}

	public void addTriggerDirectly(Trigger t) throws Exception {
		String appId = t.getAppId();
		logger.info("addTrigger() appId = " + appId);
		try {
			// check if the corresponding state monitor exists, if not, create
			// one
			StateMonitor sm = this.getStateMonitor(appId);
			if (sm == null) {
				sm = this.createStateMonitor(appId);
			}
			sm.addTrigger(t);
		} catch (Exception e) {
			throw e;
		}
	}

	private StateMonitor createStateMonitor(String appId) throws IOException {
		StateMonitor sm = monitorMap.get(appId);
		if (sm == null) {
			sm = new StateMonitor(appId);
			StateMonitor oldSM = monitorMap.putIfAbsent(appId, sm);
			if (oldSM != null) {
				sm = oldSM;
			}
		}
		return sm;
	}

	private void deleteStateMonitor(String appId) {
		monitorMap.remove(appId);
	}

	public StateMonitor getStateMonitor(String appId) {
		return monitorMap.get(appId);
	}

	public void addAppInfoPoller(String appId) {
		CFPollerManager.getInstance().addAppInfoPoller(appId);
	}

	public void addPoller(String appId) {

		// if poller is not defined as a metric source
		if (MetricConfigManager.getInstance().getEnabledMetric(getAppType(appId), Constants.METRIC_SOURCE_POLLER,
				appId) == null) {
			// if no app info record in appMetric map
			if (appMetricsMap.get(appId) == null) {
				// add a app info poller
				CFPollerManager.getInstance().addAppInfoPoller(appId);
			} else {
				// do nothing
			}
		} else {
			CFPollerManager.getInstance().addPoller(appId);
		}

	}

	public void removePoller(String appId) {

		// if poller is defined as a metric source
		if (MetricConfigManager.getInstance().getEnabledMetric(getAppType(appId), Constants.METRIC_SOURCE_POLLER,
				appId) != null) {
			CFPollerManager.getInstance().removePoller(appId);
		}
	}

	public void processAppInstanceMetrics(AppInstanceMetrics appInstanceMetrics, String dataSource /* poller */) {
		try {
			if (this.testMetricsMap.get(appInstanceMetrics.getAppId()) != null) {
				handleTestMetrics(appInstanceMetrics);
			}
			// add to application Map & DB
			addAppInstanceMetrics(appInstanceMetrics, dataSource);

			if (appInstanceMetrics.getInstanceMetrics() == null)
				return;

			// push to evaluation.
			AppInstanceMetrics clonedAppInstanceMetrics = filterAppInstanceMetricsForEvaluation(appInstanceMetrics,
					dataSource);
			if (clonedAppInstanceMetrics != null)
				metricsQueue.put(clonedAppInstanceMetrics);

		} catch (InterruptedException e) {
			logger.error(e.getMessage(), e);
		}
	}

	private AppInstanceMetrics filterAppInstanceMetricsForEvaluation(AppInstanceMetrics appInstanceMetrics,
			String dataSource) {
		String appId = appInstanceMetrics.getAppId();
		Set<String> enabledMetrics = MetricConfigManager.getInstance().getEnabledMetric(getAppType(appId), dataSource, appId);
		// if no enabled metric for this data source
		if (enabledMetrics == null) {
			return null;
		}

		List<Trigger> triggers = MonitorController.getInstance().getTriggers(appInstanceMetrics.getAppId());
		if (triggers == null || triggers.size() == 0)
			return null;

		Set<String> metricsForEvaluations = new HashSet<String>();
		for (int i = 0; i < triggers.size(); i++) {
			Trigger trigger = triggers.get(i);
			String triggerMetric = trigger.getMetric();
			if (enabledMetrics.contains(triggerMetric)) {
				if (triggerMetric.equalsIgnoreCase(Trigger.METRIC_CPU)) {
					metricsForEvaluations.add(Constants.METRIC_POLLER_CPU.toLowerCase());
				} else if (triggerMetric.equalsIgnoreCase(Trigger.METRIC_MEM)) {
					metricsForEvaluations.add(Constants.METRIC_POLLER_MEMORY_QUOATA.toLowerCase());
					metricsForEvaluations.add(Constants.METRIC_POLLER_MEMORY.toLowerCase());
				}

			} // end of if

		} // end of trigger

		if (metricsForEvaluations.size() == 0)
			return null;

		// Now, the clonedAppInstanceMetrics need to be evaluated. Clone it as a
		// new data, and get rid of the non-interested data.
		AppInstanceMetrics clonedAppInstanceMetrics = appInstanceMetrics.deepClone();
		for (InstanceMetrics instanceMetrics : clonedAppInstanceMetrics.getInstanceMetrics()) {
			for (Iterator<Metric> iter = instanceMetrics.getMetrics().iterator(); iter.hasNext();) {
				Metric metric = (Metric) iter.next();
				if (!metricsForEvaluations.contains(metric.getCompoundName())) {
					iter.remove();
				}
			}
		}

		return clonedAppInstanceMetrics;
	}

	// add AppInstanceMetrics to appMetrics Map & store to DB
	// if poller enabled, the poller metrics will be stored to DB directly
	public void addAppInstanceMetrics(AppInstanceMetrics appInstanceMetrics, String dataSource) {
		String appId = appInstanceMetrics.getAppId();
		ApplicationMetrics appMetrics = appMetricsMap.get(appId);

		if (appMetrics == null) {// no active record for this app,
			logger.debug(new StringBuilder().append("The target app ").append(appId)
					.append(" is not initialized. Add to appMetrics map now."));

			appMetrics = new ApplicationMetrics();
			appMetrics.setAppId(appInstanceMetrics.getAppId());
			appMetrics.setAppName(appInstanceMetrics.getAppName());
			appMetrics.setAppType(appInstanceMetrics.getAppType());
			appMetrics.setServiceId(appInstanceMetrics.getServiceId());
			appMetricsMap.put(appId, appMetrics);
		}

		if (dataSource.equalsIgnoreCase(Constants.METRIC_SOURCE_POLLER)) {
			appMetrics.setMemQuota(appInstanceMetrics.getMemQuota());
		}
		appMetrics.setTimestamp(System.currentTimeMillis());

		// store to db
		List<InstanceMetrics> instanceMetricsList = appInstanceMetrics.getInstanceMetrics();
		if (instanceMetricsList != null) {
			if (dataSource.equalsIgnoreCase(Constants.METRIC_SOURCE_POLLER)) {
				// for poller metrics, push to appMetricsMap first, then store
				// to db
				for (InstanceMetrics instanceMetrics : instanceMetricsList) {
					appMetrics.getPollerMetricsMap().put(instanceMetrics.getInstanceIndex(), instanceMetrics);
				}
				storeAppInstanceMetrics(appId);
			}
		}

	}

	private void storeAppInstanceMetrics(String appId) {

		try {
			// true, indicates the metrics in map will be stored to db.
			// false to indicate the stale data is not allowed.
			AppInstanceMetrics storedAppInstanceMetrics = appMetricsMap.get(appId).mergeToAppInstanceMetrics(true,
					false);
			AutoScalingDataStoreFactory.getAutoScalingDataStore().addAppStats(storedAppInstanceMetrics);

		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}

	}

	public void setStopProcessing() {
		this.processingStoped = true;
	}

	public void shutdown() {
		setStopProcessing();
		if (processingExecutor != null) {
			processingExecutor.shutdownNow();
			processingExecutor = null;
		}
		if (metricsProcessingExecutor != null) {
			metricsProcessingExecutor.shutdownNow();
			metricsProcessingExecutor = null;
		}
		if (purgeAppMetricsMapExecutor != null) {
			purgeAppMetricsMapExecutor.shutdownNow();
			purgeAppMetricsMapExecutor = null;
		}
		if (scaleProcessExecutor != null) {
			scaleProcessExecutor.shutdownNow();
			scaleProcessExecutor = null;
		}
		if (purgeLocalCacheExecutor != null) {
			purgeLocalCacheExecutor.shutdownNow();
			purgeLocalCacheExecutor = null;
		}

	}

	@Override
	public void run() {
		while (!this.processingStoped) {
			try {
				AppInstanceMetrics statsList = metricsQueue.poll(5, TimeUnit.SECONDS);
				if (statsList == null) {
					continue;
				}
				logger.debug("Taken " + statsList);
				EvaluationTask task = new EvaluationTask(statsList);
				metricsProcessingExecutor.submit(task);

			} catch (Exception e) {
				try {
					logger.error("Internal_Server_Error", e);
				} catch (Throwable ex) {
					logger.error(ex.getMessage(), ex);
				}
			}
		}
		logger.debug("ProcessMetricsForScalingThread stopped.");
	}

	public void addTestMetrics(String appId, HashMap<String, Metric> testMetricList) {
		testMetricsMap.put(appId, testMetricList);
	}

	public void removeTestMetrics(String appId) {
		testMetricsMap.remove(appId);
	}

	public void handleTestMetrics(AppInstanceMetrics appInstanceMetrics) {

		HashMap<String, Metric> testMetrics = testMetricsMap.get(appInstanceMetrics.getAppId());
		int testMetricSize = testMetrics.size();
		for (InstanceMetrics instanceMetrics : appInstanceMetrics.getInstanceMetrics()) {
			int count = 0;
			for (Metric metric : instanceMetrics.getMetrics()) {
				String key = metric.getCompoundName();
				if (testMetrics.containsKey(key)) {
					metric.setValue(testMetrics.get(key).getValue());
					count++;
				}
				if (count == testMetricSize)
					break; // break the metric loop since all matches are
							// replaced with the test value
			}
		}
	}

	public ApplicationMetrics getAppMetrics(String appId) {
		return appMetricsMap.get(appId);
	}

	public List<Trigger> getTriggers(String appId) {
		List<Trigger> triggers = null;
		StateMonitor sm = this.getStateMonitor(appId);
		if (sm != null) {
			triggers = sm.getAllTriggers();
		}
		return triggers;
	}

	class ScaleTask implements Runnable {
		private MonitorTriggerEvent event;
		private String appId;

		public ScaleTask(String appId, MonitorTriggerEvent event) {
			this.appId = appId;
			this.event = event;
		}

		@Override
		public void run() {
			try {
				ScalingEventManager.getInstance().processTriggerEvents(event);
			} catch (Exception e) {
				logger.error("Scaling task for appId " + appId + " failed with " + e.getMessage(), e);
			}
		}

	}

	class EvaluationTask implements Runnable {
		private AppInstanceMetrics appInstanceMetrics;

		public EvaluationTask(AppInstanceMetrics appInstanceMetrics) {
			this.appInstanceMetrics = appInstanceMetrics;
		}

		@Override
		public void run() {
			String appId = null;
			try {
				if (appInstanceMetrics == null) {
					return;
				}
				appId = appInstanceMetrics.getAppId();

				// get StateMonitor for this app, and only if it exists we
				// update its data
				StateMonitor sm = getStateMonitor(appId);
				if (sm != null) {
					sm.addMonitorSample(appInstanceMetrics);
					List<MonitorTriggerEvent> triggerEventList = sm.evaluateTriggers();
					if (triggerEventList != null && !triggerEventList.isEmpty()) {
						for (MonitorTriggerEvent event : triggerEventList) {
							if (ScalingEventManager.getInstance().addTriggerEvents(event)) {
								loggerEvent.debug("Submit events " + event.toString() + " to scalingProcessor");
								scaleProcessExecutor.submit(new ScaleTask(appId, event));
							} else {
								loggerEvent.debug(
										"Ignore event " + event.toString() + " as the same event type is in queue");
							}
						}
					}
				}

			} catch (Exception e) {
				logger.error("Evaluation task for " + appId + " exit with " + e.getMessage(), e);
			}
		}
	}

	public Collection<BoundApp> getSerivceBoundApps(String serviceId) {
		Collection<BoundApp> boundApps = null;
		Collection<BoundApp> boundAppsWithAppType = new LinkedList<BoundApp>();
		Map<String, BoundApp> apps = serviceBoundAppsMap.get(serviceId);
		if (apps != null) {
			boundApps = apps.values();
		}
		if (boundApps != null) {
			for (BoundApp app : boundApps) {
				boundAppsWithAppType.add(app);
			}
		}
		return boundAppsWithAppType;
	}

	public String getAppNameById(String appId) {
		String appName = null;
		BoundApp boundApp = boundAppMap.get(appId);
		if (boundApp != null) {
			appName = boundApp.getAppName();
		}
		if (appName == null) {
			appName = "";
		}

		return appName;
	}

	public void updateAppNameAndType(String appId, String appName, String appType) throws Exception {
		BoundApp boundApp = boundAppMap.get(appId);
		if (boundApp != null) {
			boundApp.setAppName(appName);
			boundApp.setAppType(appType);
			AutoScalingDataStoreFactory.getAutoScalingDataStore().updateBinding(boundApp.getServiceId(), appId, appType,
					boundApp.getAppName());
		}
	}

	public String getAppType(String appId) {
		String appType = null;

		BoundApp boundApp = boundAppMap.get(appId);
		if (boundApp != null) {
			appType = boundApp.getAppType();
		}
		if (appType == null) {
			appType = "";
		}
		return appType;
	}
	
	public void addOrUpdateBoundApp(String serviceId, String appId, String appType, String appName) {
		Map<String, BoundApp> serviceApps = serviceBoundAppsMap.get(serviceId);
		if (serviceApps == null) {
			serviceApps = new HashMap<String, BoundApp>();
			Map<String, BoundApp> oldServiceApps = serviceBoundAppsMap.putIfAbsent(serviceId, serviceApps);
			if (oldServiceApps != null) {
				serviceApps = oldServiceApps;
			}
		}
		BoundApp boundApp = new BoundApp(appId, serviceId, appType, appName);
		serviceApps.put(appId, boundApp);
		boundAppMap.put(appId, boundApp);
	}

	private void removeAppfromSerivcieAppsMap(String serviceId, String appId) {
		Map<String, BoundApp> serviceApps = serviceBoundAppsMap.get(serviceId);
		if (serviceApps != null) {
			serviceApps.remove(appId);
		}
		boundAppMap.remove(appId);
		appMetricsMap.remove(appId);
	}

	public BoundApp getBoundApp(String serviceId, String appId) {
		BoundApp app = null;
		Map<String, BoundApp> appsMap = serviceBoundAppsMap.get(serviceId);
		if (appsMap != null) {
			app = appsMap.get(appId);
		}
		return app;
	}

	public void unbindService(String serviceId, String appId) throws Exception {
		AutoScalingDataStoreFactory.getAutoScalingDataStore().removeBinding(serviceId, appId);
		if (ConfigManager.getBoolean("removeAppHistoryWithUnbind", false))
			AutoScalingDataStoreFactory.getAutoScalingDataStore().removeAppStatsWithHistory(appId);
		removeAppfromSerivcieAppsMap(serviceId, appId);
	}

	public void bindService(String serviceId, String appId) throws Exception {
		String appType = null;
		String appName = null;
		try {
			String[] appNameAndType = CloudFoundryManager.getInstance().getAppNameAndType(appId);
			appName = appNameAndType[0];
			appType = appNameAndType[1];
		} catch (Exception e) {
			logger.error("Failed to get the app type for app " + appId, e);
		}
		addOrUpdateBoundApp(serviceId, appId, appType, appName);
		AutoScalingDataStoreFactory.getAutoScalingDataStore().addBinding(serviceId, appId, appType, appName);
	}

	public void updateBoundAppName(Collection<BoundApp> appList) {
		String appId = null;
		try {
			if (null != appList) {
				for (BoundApp app : appList) {
					appId = app.getAppId();
					String[] nameAndType = CloudFoundryManager.getInstance().getAppNameAndType(appId);
					String appName = nameAndType[0];
					if (app.getAppName() != null && !app.getAppName().equals(appName)) {
						app.setAppName(appName);
						this.getAppMetrics(appId).setAppName(appName);
						AutoScalingDataStoreFactory.getAutoScalingDataStore().addBinding(app.getServiceId(), appId,
								app.getAppType(), appName);
					}
				}
			}

		} catch (Exception e) {
			logger.error("Failed to get the app type for app " + appId, e);
		}

	}

	public BoundApp getBoundApp(String appId) {
		return this.boundAppMap.get(appId);
	}

	public Map<String, Integer> getBoundAppStats() {
		Map<String, Integer> statsMap = new HashMap<String, Integer>();

		int appCount = boundAppMap.size();

		int instanceCount = 0;
		Set<String> appIds = boundAppMap.keySet();
		for (String appId : appIds) {
			ApplicationMetrics appMetrics = appMetricsMap.get(appId);
			if (appMetrics != null) {
				instanceCount += appMetrics.getPollerMetricsMap().size();
			}
		}

		statsMap.put("appCount", appCount);
		statsMap.put("instanceCount", instanceCount);

		return statsMap;
	}

	public boolean isActiveApp(String appId) {
		if (appMetricsMap.get(appId) != null)
			return true;
		else
			return false;
	}

	public void purgeAppFromMap(String appId) {
		appMetricsMap.remove(appId);
	}

}
