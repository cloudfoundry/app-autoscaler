package org.cloudfoundry.autoscaler.bean;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

/*
{
  "appId":       <appId>,
  "triggerId":   <triggerId>,
  "metricValue": <metricValue>
  "timeStamp":   <timeStamp>
}
*/

@JsonIgnoreProperties(value = { "trigger" })
public class MonitorTriggerEvent
{
	
	private String appId       = "noName";
	private String triggerId   = "0";
	private double    metricValue = 0.0;
	private String metricType;
	private long   timeStamp   = 0;
	private Trigger trigger; //The trigger that triggers this event

	public String getAppId() {
		return appId;
	}

	public void setAppId(String appId) {
		this.appId = appId;
	}

	public String getTriggerId() {
		return triggerId;
	}

	public void setTriggerId(String triggerId) {
		this.triggerId = triggerId;
	}

	public double getMetricValue() {
		return metricValue;
	}

	public void setMetricValue(double metricValue) {
		this.metricValue = metricValue;
	}

	public long getTimeStamp() {
		return timeStamp;
	}

	public void setTimeStamp(long timeStamp) {
		this.timeStamp = timeStamp;
	}

	public String getMetricType() {
		return metricType;
	}

	public void setMetricType(String metricType) {
		this.metricType = metricType;
	}

	public Trigger getTrigger() {
		return trigger;
	}

	public void setTrigger(Trigger trigger) {
		this.trigger = trigger;
	}

	
}
