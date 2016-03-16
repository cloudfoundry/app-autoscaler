package org.cloudfoundry.autoscaler;

import java.util.List;
import java.util.Map;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.bean.AutoScalerPolicyTrigger;
import org.cloudfoundry.autoscaler.bean.Trigger;
import org.cloudfoundry.autoscaler.data.couchdb.document.AutoScalerPolicy;
import org.cloudfoundry.autoscaler.exceptions.MetricNotSupportedException;
import org.cloudfoundry.autoscaler.exceptions.MonitorServiceException;
import org.cloudfoundry.autoscaler.exceptions.TriggerNotSubscribedException;
import org.cloudfoundry.autoscaler.util.IcapMonitorMetricsMapper;

/**
 * This class is used to handle subscribing triggers for a policy
 * 
 * 
 * 
 */
public class TriggerSubscriber {
	private static final Logger logger = Logger
			.getLogger(TriggerSubscriber.class.getName());
	private MonitorService monitorService;
	private static final String CALLBACK_REST_URI = "/resources/events";
	private String appId = null;
	private AutoScalerPolicy policy = null;
	public TriggerSubscriber(String appId, AutoScalerPolicy policy) throws MonitorServiceException {
		this.appId = appId;
		this.policy = policy;
		monitorService = new MonitorService(appId);
	}

	/**
	 * Subscribe triggers for a policy
	 * 
	 * @param policy
	 * @throws MetricNotSupportedException 
	 * @throws MonitorServiceException 
	 */
	public void subscribeTriggers() throws MetricNotSupportedException, MonitorServiceException {
		List<AutoScalerPolicyTrigger> policyTriggers = policy
				.getPolicyTriggers();
		if (policyTriggers == null) {
			logger.warn( "No triggers");
			return;
		}
		for (AutoScalerPolicyTrigger policyTrigger : policyTriggers) {
			subscribeTrigger(policyTrigger);
		}
	}

	/**
	 * Subscribe triggers for a policy
	 * 
	 * @param policy
	 * @throws MonitorServiceException 
	 * @throws TriggerNotSubscribedException 
	 */
	public void unSubscribeTriggers() throws MonitorServiceException, TriggerNotSubscribedException {
		unsubscribeTrigger(appId);
	}

	private void subscribeTrigger(
			AutoScalerPolicyTrigger policyTrigger) throws MonitorServiceException, MetricNotSupportedException {
		Trigger lowTrigger = createTrigger(policyTrigger,
				AutoScalerPolicyTrigger.TriggerId_LowerThreshold);
		Trigger upperTrigger = createTrigger(policyTrigger,
				AutoScalerPolicyTrigger.TriggerId_UpperThreshold);
		lowTrigger.setAppId(appId);
		upperTrigger.setAppId(appId);
		monitorService.subscribe(lowTrigger);
		monitorService.subscribe(upperTrigger);
	}

	private void unsubscribeTrigger(String appId) throws MonitorServiceException, TriggerNotSubscribedException {
		monitorService.unsubscribe(appId);
	}

	/*****************************************************************************************************************
	 * Create a trigger
	 * @throws MetricNotSupportedException 
	 */
	private Trigger createTrigger(AutoScalerPolicyTrigger policyTrigger,
			String triggerId) throws MetricNotSupportedException {
		Trigger trigger = new Trigger();

		String metricType = policyTrigger.getMetricType();
		Map<String, String> metricsMapper = IcapMonitorMetricsMapper.getMetricNameMapper();
		String metricName = metricsMapper.get(metricType.toUpperCase());
		if (metricName != null){
			trigger.setMetric(metricName);
		}else 
			throw new MetricNotSupportedException(metricType);

		 String statType = policyTrigger.getStatType();
		 if (Trigger.AGGREGATE_TYPE_MAX.equalsIgnoreCase(statType))
		 {
			 trigger.setStatType(Trigger.AGGREGATE_TYPE_MAX);
		 }
		 else {
			 trigger.setStatType(Trigger.AGGREGATE_TYPE_AVG);
		 }

		trigger.setStatWindowSecs(policyTrigger.getStatWindow());
		trigger.setBreachDurationSecs(policyTrigger.getBreachDuration());
		trigger.setUnit(policyTrigger.getUnit());

		if (triggerId.equals(AutoScalerPolicyTrigger.TriggerId_LowerThreshold)) {
			trigger.setTriggerId(AutoScalerPolicyTrigger.TriggerId_LowerThreshold);
			double threshold = IcapMonitorMetricsMapper.converMetricValue(metricName, policyTrigger.getLowerThreshold());
			trigger.setMetricThreshold(threshold);
			trigger.setThresholdType(Trigger.THRESHOLD_TYPE_LESS_THAN);
		} else if (triggerId.equals(AutoScalerPolicyTrigger.TriggerId_UpperThreshold)) {
			trigger.setTriggerId(AutoScalerPolicyTrigger.TriggerId_UpperThreshold);
			double threshold = IcapMonitorMetricsMapper.converMetricValue(metricName, policyTrigger.getUpperThreshold());
			trigger.setMetricThreshold(threshold);
			trigger.setThresholdType(Trigger.THRESHOLD_TYPE_LARGER_THAN);
		}
		trigger.setCallbackUrl(getCallbackUrl());
		return trigger;
	}
	
	private String getCallbackUrl(){
		String appUrl = AutoScalerEnv.getApplicationUrl();
		if (appUrl == null)
			appUrl = "http://localhost:8080/server";// Just for test usage, should be deleted.// TODO
		String callbackUrl = appUrl + CALLBACK_REST_URI;
		return callbackUrl;
	}
}
