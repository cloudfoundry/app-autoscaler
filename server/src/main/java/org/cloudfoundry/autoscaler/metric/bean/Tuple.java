package org.cloudfoundry.autoscaler.metric.bean;

public class Tuple {
	
	public String metricId = "";
	public double value = 0;
	public long timestamp = 0;
	public String instanceId = "";
	public double quota;
	public int evaluateCount = 0;
	
	public Tuple(String metricId, double value, long timestamp, String instanceId) {
		this.metricId = metricId;
		this.value = value;
		this.timestamp = timestamp;
		this.instanceId = instanceId;
		this.evaluateCount = 0;
	}

	public Tuple(String metricId, double value, long timestamp, String instanceId, double quota) {
		this.metricId = metricId;
		this.value = value;
		this.timestamp = timestamp;
		this.instanceId = instanceId;
		this.quota = quota;
		this.evaluateCount = 0;
	}
	
	public String getMetricId() {
		return metricId;
	}
	
	public void setMetricId(String metricId) {
		this.metricId = metricId;
	}
	
	public double getValue() {
		return value;
	}
	
	public void setValue(double value) {
		this.value = value;
	}
	
	public long getTimestamp() {
		return timestamp;
	}
	
	public void setTimestamp(long timestamp) {
		this.timestamp = timestamp;
	}
	
	public String getInstanceId() {
		return instanceId;
	}
	
	public void setInstanceId(String instanceId) {
		this.instanceId = instanceId;
	}
	
	public String toString() {
		String output = "InstanceId ["+this.instanceId+"] MetricId ["+this.metricId+"] " +
				"Value ["+this.value+"] Timestamp ["+this.timestamp+"]";
		return output;
	}

	public double getQuota() {
		return quota;
	}

	public void setQuota(double quota) {
		this.quota = quota;
	}

	public int getEvaluateCount() {
		return evaluateCount;
	}

	public void setEvaluateCount(int evaluateCount) {
		this.evaluateCount = evaluateCount;
	}

	public void increaseEvaluateCount() {
		this.evaluateCount ++ ;
	}

	
	
}
