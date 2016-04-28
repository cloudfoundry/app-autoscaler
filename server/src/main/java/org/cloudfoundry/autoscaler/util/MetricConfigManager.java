package org.cloudfoundry.autoscaler.util;

import java.util.HashMap;
import java.util.HashSet;
import java.util.Iterator;
import java.util.Map;
import java.util.Map.Entry;
import java.util.Set;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.bean.Trigger;
import org.cloudfoundry.autoscaler.common.util.ConfigManager;
import org.cloudfoundry.autoscaler.constant.Constants;
import org.cloudfoundry.autoscaler.metric.monitor.MonitorController;
import org.cloudfoundry.autoscaler.metric.monitor.StateMonitor;

import com.fasterxml.jackson.databind.ObjectMapper;

public class MetricConfigManager {
	// private static final int DEFAULT_TIMEOUT = 120;

	private static final Logger logger = Logger.getLogger(MetricConfigManager.class);

	// definition: appType, Map<> Configuration
	// or appType # appId, Map<> Configuration
	private Map<String, Map<String, Object>> defaultConfigMap = null;

	// definition: appType, Set<String> metricList
	// or appType # appId, Set<String> metricList
	private Map<String, Set<String>> defaultEnableMetricMap = null;
	// [poller]
	private String[] metricPriorityArray = ConfigManager.get(Constants.METRIC_PREFIX + Constants.METRIC_SEPERATOR
			+ Constants.METRIC_SOURCE + Constants.METRIC_SEPERATOR + "priority", "poller").replaceAll("\\s*", "")
			.split(",");

	private MonitorController controller = MonitorController.getInstance();
	private ObjectMapper mapper = new ObjectMapper();
	private static MetricConfigManager instance = new MetricConfigManager();

	private MetricConfigManager() {
	}

	public static MetricConfigManager getInstance() {
		return instance;
	}

	public Map<String, Object> loadDefaultConfig(String appType, String appId) {

		Map<String, Object> defaultConfig = loadDefaultConfig(appType);
		if (appId == null)
			return defaultConfig;

		boolean isCPURequired = isCPURequired(appId);
		;
		boolean isPollerMemRequired = isPollerMemRequired(appId);

		String key = appType;
		if (isCPURequired || isPollerMemRequired) {
			key = new StringBuilder().append(appType).append("#").append(appId).toString();
		} else
			return defaultConfig;

		HashMap<String, Object> customConfig = (HashMap<String, Object>) ((HashMap<String, Object>) defaultConfig)
				.clone();
		for (Entry<String, Object> entry : customConfig.entrySet()) {
			Object value = entry.getValue();
			if (value instanceof HashMap<?, ?>) {
				HashMap<String, Set<String>> clonedValue = (HashMap<String, Set<String>>) ((HashMap<?, ?>) value)
						.clone();
				for (Entry<String, Set<String>> innerEntry : clonedValue.entrySet()) {
					HashSet<String> innerValue = new HashSet<String>();
					innerValue.addAll(innerEntry.getValue());
					clonedValue.put(innerEntry.getKey(), innerValue);
				}
				customConfig.put(entry.getKey(), clonedValue);
			}
		}

		HashMap<String, Set<String>> metricMap = (HashMap<String, Set<String>>) customConfig
				.get(Constants.METRICS_CONFIG);
		for (Entry<String, Set<String>> entry : metricMap.entrySet()) {
			Set<String> enabledMetric = entry.getValue();
			if (isCPURequired) {
				if (enabledMetric != null) {
					enabledMetric.add(Trigger.METRIC_CPU);
				} else {
					try {
						logger.debug(new StringBuilder().append("Wrong config item for app ").append(appId)
								.append(mapper.writeValueAsString(customConfig)));
					} catch (Exception e) {
						logger.info(e.getMessage());
					}
				}
			}
		}

		if (isCPURequired) {

			Set<String> pollerMetrics = metricMap.get(Constants.METRIC_SOURCE_POLLER);
			if (pollerMetrics != null) {
				pollerMetrics.add(Trigger.METRIC_CPU);
			}
		}

		if (isPollerMemRequired) {
			Set<String> pollerMetrics = metricMap.get(Constants.METRIC_SOURCE_POLLER);
			if (pollerMetrics == null)
				pollerMetrics = new HashSet<String>();
			pollerMetrics.add(Trigger.METRIC_MEM);
			metricMap.put(Constants.METRIC_SOURCE_POLLER, pollerMetrics);

		}

		// add to defaultMetricList for quick search
		for (Entry<String, Set<String>> entry : metricMap.entrySet()) {
			defaultEnableMetricMap.put(key + "#" + entry.getKey(), entry.getValue());
		}

		defaultConfigMap.put(key, customConfig);

		try {
			logger.debug(new StringBuilder().append("New config item for app ").append(appId)
					.append(mapper.writeValueAsString(customConfig)));
		} catch (Exception e) {
			logger.info(e.getMessage());
		}

		return customConfig;
	}

	public Map<String, Object> loadDefaultConfig(String appType) {

		if (defaultConfigMap == null) {
			defaultConfigMap = new HashMap<String, Map<String, Object>>();
		}
		if (defaultEnableMetricMap == null) {
			defaultEnableMetricMap = new HashMap<String, Set<String>>();
		}

		if (appType == null)
			return null;

		String key = appType;

		if (defaultConfigMap.get(key) == null) {

			Map<String, Object> configMap = new HashMap<String, Object>();
			// handle common configuration
			configMap.put(Constants.REPORT_INTERVAL, ConfigManager.getInt(Constants.REPORT_INTERVAL, 60));
			// metricSource poller

			// poller
			String metricSource = ConfigManager.get(Constants.METRIC_PREFIX + Constants.METRIC_SEPERATOR + appType,
					ConfigManager
							.get(Constants.METRIC_PREFIX + Constants.METRIC_SEPERATOR + Constants.METRIC_WILDCARD));

			if (metricSource == null) {
				logger.error("ERROR: missing metric source configuration for key " + key);
				defaultConfigMap.put(key, configMap);
				return configMap;
			} else {
				metricSource = metricSource.replaceAll("\\s*", "");
			}

			String[] metricSourceArray = metricSource.split(",");
			Map<String, Set<String>> metricMap = new HashMap<String, Set<String>>();
			// source, poller
			for (String source : metricSourceArray) {
				// get rid of invalid data source setting
				if (source.isEmpty() || !source.equalsIgnoreCase(Constants.METRIC_SOURCE_POLLER))
					continue;

				boolean enabled = ConfigManager.getBoolean(
						Constants.METRIC_PREFIX + Constants.METRIC_SEPERATOR + appType + Constants.METRIC_SEPERATOR
								+ source + Constants.METRIC_SEPERATOR + Constants.METRIC_SOURCE_STATUS,
						ConfigManager.getBoolean(Constants.METRIC_PREFIX + Constants.METRIC_SEPERATOR
								+ Constants.METRIC_WILDCARD + Constants.METRIC_SEPERATOR + source
								+ Constants.METRIC_SEPERATOR + Constants.METRIC_SOURCE_STATUS, true));

				// if a datasource is disabled, ignore it
				if (!enabled)
					continue;

				String enabledMetricString = ConfigManager.get(
						Constants.METRIC_PREFIX + Constants.METRIC_SEPERATOR + appType + Constants.METRIC_SEPERATOR
								+ source + Constants.METRIC_SEPERATOR + Constants.METRIC_SOURCE_TYPE,
						ConfigManager.get(Constants.METRIC_PREFIX + Constants.METRIC_SEPERATOR
								+ Constants.METRIC_WILDCARD + Constants.METRIC_SEPERATOR + source
								+ Constants.METRIC_SEPERATOR + Constants.METRIC_SOURCE_TYPE, ""));

				// if no metric defined for a datasource, ignore it.
				if (enabledMetricString == null)
					continue;

				String[] enabledMetricStringArray = enabledMetricString.replaceAll("\\s*", "").split(",");
				Set<String> enabledMetric = new HashSet<String>();
				for (String metric : enabledMetricStringArray) {
					if (!metric.isEmpty()) {
						if (metric.equalsIgnoreCase(Trigger.METRIC_CPU))
							enabledMetric.add(Trigger.METRIC_CPU);
						else if (metric.equalsIgnoreCase(Trigger.METRIC_MEM))
							enabledMetric.add(Trigger.METRIC_MEM);
						else
							enabledMetric.add(metric);
					}
				}

				// if no valid metric defined for a datasource, ignore it.
				if (enabledMetric.size() == 0)
					continue;

				metricMap.put(source, enabledMetric);

				// <poller,[Memory]>
			}

			Set<String> allEnabledMetrics = new HashSet<String>();
			for (int i = 0; i < metricPriorityArray.length; i++) {
				Set<String> enabledMetric = metricMap.get(metricPriorityArray[i]);
				if (enabledMetric == null)
					continue;

				if (i == 0) {
					allEnabledMetrics.addAll(enabledMetric);
					continue;
				}
				for (Iterator<String> iter = enabledMetric.iterator(); iter.hasNext();) {
					String metric = (String) iter.next();
					if (allEnabledMetrics.contains(metric)) {
						iter.remove();
					} else {
						allEnabledMetrics.add(metric);
					}
				}
				if (enabledMetric.size() == 0)
					metricMap.remove(metricPriorityArray[i]);
			}

			// add to defaultMetricList for quick search
			for (Entry<String, Set<String>> entry : metricMap.entrySet()) {
				defaultEnableMetricMap.put(key + "#" + entry.getKey(), entry.getValue());
			}

			configMap.put(Constants.METRICS_CONFIG, metricMap);
			defaultConfigMap.put(key, configMap);
		}

		return defaultConfigMap.get(key);

	}

	public Set<String> getEnabledMetric(String appType, String dataSource, String appId) {

		if (dataSource == null || appType == null)
			return null;

		boolean isCPURequired = false;
		if (appId != null) {
			isCPURequired = isCPURequired(appId);
		}
		boolean isPollerMemRequired = false;
		if (appId != null) {
			isPollerMemRequired = isPollerMemRequired(appId);
		}

		String key = isCPURequired || isPollerMemRequired
				? new StringBuilder().append(appType).append("#").append(appId).toString() : appType;
		if ((defaultEnableMetricMap == null) || (defaultEnableMetricMap.get(key + "#" + dataSource) == null)) {
			// not initialized?
			loadDefaultConfig(appType, appId);
		}

		Set<String> enabledMetric = defaultEnableMetricMap.get(key + "#" + dataSource);
		try {
			logger.debug(new StringBuilder().append("Enabled metric for ").append(appId).append(" with datasource ")
					.append(dataSource).append(" : ").append(mapper.writeValueAsString(enabledMetric)));
		} catch (Exception e) {
			logger.info(e.getMessage());
		}
		return enabledMetric;

	}

	private boolean isCPURequired(String appId) {
		if (appId.equalsIgnoreCase("testID"))
			return true;

		StateMonitor stateMonitor = controller.getStateMonitor(appId);
		return (stateMonitor != null) ? stateMonitor.isCPURequired() : false;
	}

	private boolean isPollerMemRequired(String appId) {
		if (appId.equalsIgnoreCase("testID"))
			return true;

		StateMonitor stateMonitor = controller.getStateMonitor(appId);
		return (stateMonitor != null) ? stateMonitor.isPollerMemRequired() : false;
	}

}
