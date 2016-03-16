package org.cloudfoundry.autoscaler.bean;

import java.io.Serializable;
import java.util.ArrayList;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

/**
 * To remain backward compatibility, the monitor checks whether conditionList is empty. If empty, it uses the condition
 * defined in the fields of Tigger, If not, it ignores the condition defined in the fields of Trigger but uses those
 * defined in the conditionList
 * 
 *
 * 
 */

@JsonIgnoreProperties(ignoreUnknown = true)
public class Trigger implements Serializable {
    /**
	 * 
	 */
	private static final long serialVersionUID = 1L;
	public static final String METRIC_CPU = "CPU";
    public static final String METRIC_MEM = "Memory";

    public static final String THRESHOLD_TYPE_LARGER_THAN = "larger_than";
    public static final String THRESHOLD_TYPE_LESS_THAN = "less_than";

    public static final String AGGREGATE_TYPE_AVG = "avg";

    public static final String AGGREGATE_TYPE_MAX = "max";

    private String appName = "";
    private String appId = "";
    private String triggerId = "0";
    private String metric = METRIC_CPU;
    private String windowStatType = AGGREGATE_TYPE_AVG;
    private int statWindowSecs = 60;
    private int breachDurationSecs = 60;
    private double metricThreshold = 70;
    private String thresholdType = THRESHOLD_TYPE_LARGER_THAN;
    private String callbackUrl = "";
    private String unit = "percent";

    public ArrayList<Condition> conditionList = null;

    public Trigger() {
        this.conditionList = new ArrayList<Condition>();
    }

    public void addCondition(Condition c) {
        this.conditionList.add(c);
    }

    public String toString() {
        String output = "Trigger [" + this.triggerId + "] for App [" + this.getAppId() + "]\n";
        output += "MetricId [" + this.metric + "] Threshold [" + this.metricThreshold + "] BreachDuration ["
                + this.breachDurationSecs + "]\n";
        return output;
    }

    public static enum ThresholdUnit {
        PERCENT, MB
    }

    public String getAppName() {
        return appName;
    }

    public void setAppName(String appName) {
        this.appName = appName;
    }

    public String getTriggerId() {
        return triggerId;
    }

    public void setTriggerId(String triggerId) {
        this.triggerId = triggerId;
    }

    public String getMetric() {
        return metric;
    }

    public void setMetric(String metric) {
        this.metric = metric;
    }

    public String getStatType() {
        return windowStatType;
    }

    public void setStatType(String statType) {
        this.windowStatType = statType;
    }

    public int getStatWindowSecs() {
        return statWindowSecs;
    }

    public void setStatWindowSecs(int statWindowSecs) {
        this.statWindowSecs = statWindowSecs;
    }

    public int getBreachDurationSecs() {
        return breachDurationSecs;
    }

    public void setBreachDurationSecs(int breachDurationSecs) {
        this.breachDurationSecs = breachDurationSecs;
    }

    public double getMetricThreshold() {
        return metricThreshold;
    }

    public void setMetricThreshold(double metricThreshold) {
        this.metricThreshold = metricThreshold;
    }

    public String getThresholdType() {
        return thresholdType;
    }

    public void setThresholdType(String thresholdType) {
        this.thresholdType = thresholdType;
    }

    public String getCallbackUrl() {
        return callbackUrl;
    }

    public void setCallbackUrl(String callbackUrl) {
        this.callbackUrl = callbackUrl;
    }

    public String getUnit() {
        return unit;
    }

    public void setUnit(String unit) {
        this.unit = unit;
    }

    public String getAppId() {
        return appId;
    }

    public void setAppId(String appId) {
        this.appId = appId;
    }

    public String generateKey(){
    	return this.getMetric() +  "_" + this.getTriggerId() + "_" + this.getMetricThreshold();
    }
}
