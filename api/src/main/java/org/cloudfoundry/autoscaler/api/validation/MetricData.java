package org.cloudfoundry.autoscaler.api.validation;

import java.util.List;

import javax.validation.Valid;
import javax.validation.constraints.NotNull;

public class MetricData {
	@NotNull(message="{MetricData.appId.NotNull}")
	private String appId;

	@NotNull(message="{MetricData.appName.NotNull}")
	private String appName;

	@NotNull(message="{MetricData.appType.NotNull}")
	private String appType;

	@NotNull(message="{MetricData.timestamp.NotNull}")
    private long timestamp;

	@Valid
	private List<InstanceMetric> instanceMetrics;

	public String getAppId(){
		return this.appId;
	}

	public void setAppId(String appId){
		this.appId = appId;
	}

	public String getAppName() {
		return this.appName;
	}

	public void setAppName(String appName) {
		this.appName = appName;
	}

	public String getAppType() {
		return this.appType;
	}

	public void setAppType(String appType) {
		this.appType = appType;
	}

	public List<InstanceMetric> getInstanceMetrics() {
		return this.instanceMetrics;
	}

	public void setInstanceMetrics(List<InstanceMetric> instanceMetrics) {
		this.instanceMetrics = instanceMetrics;
	} 

    public long getTimestamp() {
    	return this.timestamp;
    }

    public void setTimestamp(long timestamp){
    	this.timestamp = timestamp;
    }

}