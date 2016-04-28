package org.cloudfoundry.autoscaler.metric.monitor;

import java.util.ArrayList;
import java.util.Calendar;
import java.util.Collection;
import java.util.HashMap;
import java.util.HashSet;
import java.util.Iterator;
import java.util.LinkedList;
import java.util.List;
import java.util.Map.Entry;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.bean.Condition.AggregationType;
import org.cloudfoundry.autoscaler.bean.InstanceMetrics;
import org.cloudfoundry.autoscaler.bean.Metric;
import org.cloudfoundry.autoscaler.bean.MonitorTriggerEvent;
import org.cloudfoundry.autoscaler.bean.Trigger;
import org.cloudfoundry.autoscaler.bean.Trigger.ThresholdUnit;
import org.cloudfoundry.autoscaler.common.util.ConfigManager;
import org.cloudfoundry.autoscaler.constant.Constants;
import org.cloudfoundry.autoscaler.data.couchdb.document.AppInstanceMetrics;
import org.cloudfoundry.autoscaler.metric.bean.Statistic;
import org.cloudfoundry.autoscaler.metric.bean.Tuple;

/**
 * A state monitor is created for each application This class needs to be
 * thread-safe.
 * 
 * @author smeng
 *
 */
public class StateMonitor {

	private static final Logger logger = Logger.getLogger(StateMonitor.class);

	public static final int TRIGGER_CHECK_INTERVAL_IN_SEC = 10;
	public static final String strSeparator = "-##@@##-";

	public static String autoScalerURL = (System.getenv("autoscalerURL") != null) ? System.getenv("autoscalerURL")
			: "http://localhost:9080/autoscaler";

	private String appId;

	// a hashmap with key defined by <instanceId>-##@@##-<metricId>
	private HashMap<String, LinkedList<Tuple>> instanceMetricTupleMap = null;

	// TODO: replace this with the global naming service
	private HashSet<String> instanceSet = null;

	private HashMap<String, Trigger> triggerMap;
	private HashMap<String, Long> breachCounterMap;

	// temporary usage to disable CPU metrics
	private boolean isCPURequired;
	private boolean isPollerMemRequired;

	private long prevTriggerEvalTime = 0;

	public StateMonitor(String appId) {
		this.appId = appId;
		this.instanceMetricTupleMap = new HashMap<String, LinkedList<Tuple>>();
		this.instanceSet = new HashSet<String>();

		triggerMap = new HashMap<String, Trigger>();
		breachCounterMap = new HashMap<String, Long>();

		isCPURequired = false;
		isPollerMemRequired = false;

	}

	public String getAppId() {
		return appId;
	}

	public int getNumTriggers() {
		return triggerMap.size();
	}

	public long getPrevTriggerEvalTime() {
		return prevTriggerEvalTime;
	}

	public void addMonitorSample(AppInstanceMetrics appInstanceMetrics) {

		List<InstanceMetrics> instanceMetricsList = appInstanceMetrics.getInstanceMetrics();

		for (InstanceMetrics instanceMetrics : instanceMetricsList) {
			String instanceIndex = String.valueOf(instanceMetrics.getInstanceIndex());
			List<Metric> metrics = instanceMetrics.getMetrics();
			double usedMemory = 0; // Unit is MB
			double maxMemory = 0;// Unit is MB
			for (Metric metric : metrics) {

				String compoundName = metric.getCompoundName().toLowerCase();

				// poller cpu
				if (compoundName.equalsIgnoreCase(Constants.METRIC_POLLER_CPU)) {
					this.addTuple(new Tuple(Trigger.METRIC_CPU, Double.valueOf(metric.getValue()) * 100,
							metric.getTimestamp(), instanceIndex));
					continue;
				}

				if (compoundName.contains("memory")) {

					maxMemory = appInstanceMetrics.getMemQuota();

					if (compoundName.equalsIgnoreCase(Constants.METRIC_POLLER_MEMORY)) {
						usedMemory = Double.valueOf(metric.getValue());
					} else
						continue; // ignore the other memory related input;
					if ((usedMemory > 0) && (maxMemory > 0)) {
						this.addTuple(new Tuple(Trigger.METRIC_MEM, usedMemory, metric.getTimestamp(), instanceIndex,
								maxMemory));
					}
					continue;
				}

			}
		}

	}

	public void addTuple(Tuple tp) {
		synchronized (this.instanceMetricTupleMap) {

			String key = tp.getInstanceId() + strSeparator + tp.getMetricId();
			LinkedList<Tuple> tupleList = this.instanceMetricTupleMap.get(key);

			if (null == tupleList) {
				logger.debug("Create tuple list for key [" + key + "]");
				tupleList = new LinkedList<Tuple>();
				this.instanceMetricTupleMap.put(key, tupleList);
			}

			logger.debug(
					"Adding a tuple [" + tp.toString() + "] to state monitor. Key is " + key + ". AppId is " + appId);
			int maxTimeToKeepInSec = this.getMaxTimeToKeepInSec();
			synchronized (tupleList) {

				// remove expired data point if oversized,
				if (tupleList.size() > maxTimeToKeepInSec) {

					long curTime = Calendar.getInstance().getTimeInMillis();
					Iterator<Tuple> iterator = tupleList.iterator();
					Tuple oneTuple = iterator.next();

					if ((curTime - maxTimeToKeepInSec * 1000) > oneTuple.getTimestamp()) {
						// expired tuple
						logger.debug("Removing one expired tuple [" + oneTuple.toString() + "]");
						iterator.remove();
					}
				}

				tupleList.add(tp);
			}
			logger.debug("TupleList for key [" + key + "] size [" + tupleList.size() + "]. AppId is " + appId);
		}
		synchronized (this.instanceSet) {
			// TODO: remove this after having global naming service
			this.instanceSet.add(tp.getInstanceId());
		}

	}

	public synchronized void addTrigger(Trigger t) {
		String key = t.generateKey();
		// if the trigger already exists we don't do anything (no need to throw
		// an exception)
		if (triggerMap.containsKey(key)) {
			logger.warn("Attempting to add a duplicate trigger, ID = " + t.getTriggerId());
			return;
		}

		// add the trigger to the list of triggers and the breach counter list
		triggerMap.put(key, t);
		if (t.getMetric().equalsIgnoreCase(Trigger.METRIC_CPU)) {
			isCPURequired = true;
		}
	}

	public synchronized List<Trigger> getAllTriggers() {
		Collection<Trigger> triggerColl = triggerMap.values();
		List<Trigger> triggers = new ArrayList<Trigger>(triggerColl.size());
		Iterator<Trigger> it = triggerColl.iterator();
		while (it.hasNext()) {
			triggers.add(it.next());
		}
		return triggers;
	}

	public synchronized void removeTrigger(Trigger t) {
		String key = t.generateKey();
		// if the trigger does not exists we don't do anything (no need to throw
		// an exception)
		if (!triggerMap.containsKey(key)) {
			logger.warn("Attempting to remove a non-existing trigger, ID = " + t.getTriggerId());
			return;
		}

		// remove the trigger from the list of triggers and the breach counter
		// list
		triggerMap.remove(key);
		breachCounterMap.remove(key);
		if (t.getMetric().equalsIgnoreCase(Trigger.METRIC_CPU)) {
			isCPURequired = false;
		}

	}

	public boolean isCPURequired() {
		return isCPURequired;
	}

	public boolean isPollerMemRequired() {
		return isPollerMemRequired;
	}

	public void setPollerMemRequired(boolean required) {
		isPollerMemRequired = required;
	}

	public synchronized List<MonitorTriggerEvent> evaluateTriggers() {
		logger.debug("Evaluating Triggers for " + appId);

		HashSet<String> inactiveInstanceList = new HashSet<String>();
		ArrayList<MonitorTriggerEvent> eventList = new ArrayList<MonitorTriggerEvent>();
		synchronized (this.triggerMap) {
			synchronized (this.instanceMetricTupleMap) {
				for (Trigger t : this.triggerMap.values()) {
					long curTime = Calendar.getInstance().getTimeInMillis();
					int statWindow = t.getStatWindowSecs();
					double quota = 0;
					AggregationType statType = AggregationType.valueOf(t.getStatType().toUpperCase());// stat
																										// type,
																										// MAX
																										// or
																										// AVG
					Statistic globalStat = new Statistic(statType); // globalStat
																	// is used
																	// calculate
																	// and store
																	// statistic
																	// value of
																	// this app

					if (this.instanceSet.isEmpty()) {
						logger.warn("No monitoring data avaiable for trigger [" + t.getMetric() + ":" + t.getTriggerId()
								+ "] with appId " + appId + ". Possile wrong appId specified in the trigger");
					}
					// Check the tuples of each instance of the app
					for (String instanceId : this.instanceSet) {

						Statistic instanceStat = new Statistic(statType);
						String metricId = t.getMetric();
						String key = instanceId + strSeparator + metricId;

						logger.debug("Retreving tuples for app " + appId + " with key " + key);
						LinkedList<Tuple> tl = this.instanceMetricTupleMap.get(key);

						if (null == tl || tl.size() == 0) {
							logger.warn("No tuples found for app: " + appId + " for trigger: key=" + key
									+ " , thresholdType=" + t.getThresholdType());
							continue;
						}

						logger.debug("Found [" + tl.size() + "] tuples for app " + appId + " for trigger: key=" + key
								+ " , thresholdType=" + t.getThresholdType());

						// get the metric quota
						quota = tl.get(0).getQuota();
						int maxTimeToKeepInSec = this.getMaxTimeToKeepInSec();
						synchronized (tl) {
							Iterator<Tuple> iterator = tl.iterator();
							while (iterator.hasNext()) {
								Tuple tp = iterator.next();
								/** Check if the tuple is valid **/
								if (isValidTuple(tp, statWindow, curTime, maxTimeToKeepInSec)) {
									tp.increaseEvaluateCount();
									double value = tp.getValue();
									/**
									 * calculate the statistic value of this
									 * instance
									 **/
									instanceStat.update(value);
									logger.debug("update instanceStat for app " + appId + " for instance : "
											+ instanceId + "  with value " + instanceStat.getValue());
								} else
									iterator.remove();
							}
						}

						if (instanceStat.getCount() > 0) {
							logger.debug("[" + instanceStat.getCount() + "] valid tuples for key [" + key + "] :"
									+ t.getThresholdType() + " for app " + appId + " instanceId " + instanceId);
							/** Calculate the statistic value of the metric **/
							globalStat.update(instanceStat.getValue());
							logger.debug("update glabalStat for app " + appId + " with value " + globalStat.getValue()
									+ " with count " + globalStat.getCount());
							// avoid marking an instance as inactive just
							// because it fails to report some (not all) metric
							// values
							inactiveInstanceList.remove(instanceId);
						} else {
							logger.debug("No valid tuple could be used for key [" + key + "], adding instanceId ["
									+ instanceId + "] to inactive list");
							// the instance is not sending any data within the
							// statWindow, remove the instance
							inactiveInstanceList.add(instanceId);
						}

						logger.debug("Aggreated [" + t.getMetric() + "] for instance [" + instanceId + "] in app ["
								+ this.appId + "]: " + instanceStat.getValue());
					}

					if (globalStat.getCount() == 0)
						continue;

					// If the all the metric statistic values reaches the
					// threshold during the breach duration, fire a event
					if (shouldFireEvent(t, quota, globalStat)) {
						MonitorTriggerEvent event = createEvent(t, globalStat);
						eventList.add(event);
						logger.debug("Create an event " + event.toString() + " for app " + t.getAppId()
								+ ". Threshold type is " + t.getTriggerId());
					}
				}
			}
		}
		/** remove inactive instances **/
		removeInactiveInstances(inactiveInstanceList);

		prevTriggerEvalTime = System.currentTimeMillis();
		return eventList;
	}

	/**
	 * Checks if should fire an event
	 * 
	 * @param t
	 * @param quota
	 * @param globalStat
	 * @return true if should
	 */
	private boolean shouldFireEvent(Trigger t, double quota, Statistic globalStat) {

		double threshold = getThesholdByUnit(quota, t.getMetricThreshold(), t.getUnit());
		logger.debug("Aggreated [" + t.getMetric() + "-" + t.getThresholdType() + "] on all instances in app ["
				+ this.appId + "]: " + globalStat.getValue() + " with the threshold: " + threshold);
		// If this trigger is a upper threshold
		if (t.getThresholdType().equals(Trigger.THRESHOLD_TYPE_LARGER_THAN)) {
			// Check if the metric value is greater than threshold
			if (globalStat.getValue() >= threshold) {
				if (checkBreachDuration(threshold, t)) {
					logger.debug("Start counting for app " + t.getAppId() + " with trigger " + t.getTriggerId());
					return true;
				}
			} else {
				setBreachStartTime(t, null);
			}

		} else if (t.getThresholdType().equals(Trigger.THRESHOLD_TYPE_LESS_THAN)) {
			if (globalStat.getValue() <= threshold) {
				if (this.checkBreachDuration(threshold, t)) {
					logger.debug("Start counting for app " + t.getAppId() + " with trigger " + t.getTriggerId());
					return true;
				}
			} else {
				setBreachStartTime(t, null);
			}
		}
		return false;
	}

	/** Checks if a tuple is valid when evaluate triggers **/
	private boolean isValidTuple(Tuple tp, int statWindow, long curTime, long maxTimeToKeepInSec) {

		if (tp.getValue() < 0)
			return false;

		logger.debug("Checking tuple [" + tp.toString() + "], " + "current timestamp [" + curTime + "], "
				+ "time difference [" + (curTime - tp.timestamp) / 1000 + "] s");
		/** remove expired tuples **/
		if ((curTime - maxTimeToKeepInSec * 1000) > tp.getTimestamp()) {
			// expired tuple
			logger.debug("Removing an expired tuple [" + tp.toString() + "], current timestamp [" + curTime + "]");
			return false;
			/** check if the tuple is in the stat window **/
		} else if ((curTime - tp.getTimestamp()) > statWindow * 1000 + 1) {
			// tuple within the statistic window
			if (tp.getEvaluateCount() < 2) // the min value of evaluation count
											// should be 2
				logger.warn("Delete un-evaluated tuple [" + tp.toString() + "] of application " + appId
						+ " with current timestamp [" + curTime + "] while the evaluation count is "
						+ tp.getEvaluateCount());
			else
				logger.debug("Expired tuple [" + tp.toString() + "]  with current timestamp [" + curTime
						+ "] while the evaluation count is " + tp.getEvaluateCount());
			return false;
		}
		return true;
	}

	public void stopMonitor() {
	}

	/**
	 * Check if the metric value is greater than the upper threshold or less
	 * than the lower threshold for the breach duration
	 * 
	 * @param threshold
	 * @param t
	 * @return true if it lasts for the breach duration
	 */
	private boolean checkBreachDuration(double threshold, Trigger t) {
		long expectedBreachDuration = t.getBreachDurationSecs() * 1000L;
		Long breachStartTime = this.breachCounterMap.get(t.generateKey());
		if (null == breachStartTime) {
			setBreachStartTime(t, Long.valueOf(System.currentTimeMillis()));
			logger.debug("Set breach start time to NOW for app " + t.getAppId() + ", trigger " + t.getTriggerId() + "-"
					+ t.getMetric() + "-" + t.getThresholdType());
			return false;
		}
		long currentTime = System.currentTimeMillis();
		long breachDuration = currentTime - breachStartTime.longValue();
		logger.debug("Current breach duration is " + breachDuration / 1000 + " seconds. Expected breach duration is "
				+ t.getBreachDurationSecs());
		if (breachDuration >= expectedBreachDuration) {
			return true;
		} else
			return false;
	}

	/**
	 * Resets breach start time
	 * 
	 * @param t
	 */
	private void setBreachStartTime(Trigger t, Long time) {
		this.breachCounterMap.put(t.generateKey(), time);

	}

	/**
	 * Creates an event and add it to the event list.
	 * 
	 * @param t
	 * @param globalStat
	 * @param eventList
	 */
	private MonitorTriggerEvent createEvent(Trigger t, Statistic globalStat) {
		MonitorTriggerEvent event = new MonitorTriggerEvent();
		event.setAppId(this.appId);
		event.setTriggerId(t.getTriggerId());
		event.setMetricValue(globalStat.getValue());
		event.setTimeStamp(System.currentTimeMillis());
		event.setTrigger(t);
		event.setMetricType(t.getMetric());
		return event;
	}

	private void removeInactiveInstances(HashSet<String> inactiveInstanceList) {
		for (String hostname : inactiveInstanceList) {
			logger.debug("Removing instance [" + hostname + "] from state monitor since it's inactive.");
			synchronized (this.instanceSet) {
				this.instanceSet.remove(hostname);
			}
			synchronized (this.instanceMetricTupleMap) {
				Iterator<Entry<String, LinkedList<Tuple>>> iter = this.instanceMetricTupleMap.entrySet().iterator();
				while (iter.hasNext()) {
					String key = iter.next().getKey();
					if (key.contains(hostname)) {
						iter.remove();
					}
				}
			}
		}
	}

	/**
	 * Gets threshold by unit
	 * 
	 * @param quota
	 * @param threshold
	 * @param unit
	 * @return
	 */
	private double getThesholdByUnit(double quota, double threshold, String unit) {
		if (quota == 0)
			return threshold;
		ThresholdUnit thresholdUnit = ThresholdUnit.valueOf(unit.toUpperCase());
		double realThreshold = 0;
		switch (thresholdUnit) {
		case PERCENT:
			realThreshold = (double) (quota * threshold) / 100;
			break;
		default:
			realThreshold = threshold;
			break;

		}
		return realThreshold;
	}

	private int getMaxTimeToKeepInSec() {
		int time = ConfigManager.getInt("MAX_TIME_TO_KEEP_METRICS_IN_SEC");
		if (time == 0)
			time = 600;// default
		return time;
	}

}
