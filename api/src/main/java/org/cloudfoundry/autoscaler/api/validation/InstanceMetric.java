package org.cloudfoundry.autoscaler.api.validation;

import java.util.LinkedList;
import java.util.List;

import javax.validation.Valid;
import javax.validation.constraints.NotNull;

public class InstanceMetric {
	@NotNull( message="{InstanceMetric.instanceIndex.NotNull}")
    private int instanceIndex;

	@NotNull( message="{InstanceMetric.timestamp.NotNull}")
	private long timestamp;

	@NotNull( message="{InstanceMetric.instanceId.NotNull}")
    private String instanceId;

	@NotNull( message="{InstanceMetric.metrics.NotNull}")
	@Valid
    private List<Metric> metrics = new LinkedList<Metric>();

    public int getInstanceIndex() {
    	return this.instanceIndex;
    }

    public void setInstanceIndex(int instanceIndex) {
    	this.instanceIndex = instanceIndex;
    }

	public long getTimestamp() {
		return this.timestamp;
	}

	public void setTimestamp(long timestamp) {
		this.timestamp = timestamp;
	}

    public String getInstanceId() {
    	return this.instanceId;
    }

    public void setInstanceId(String instanceId) {
    	this.instanceId = instanceId;
    }

    public List<Metric> getMetrics() {
    	return this.metrics;
    }

    public void setMetrics(List<Metric> metrics) {
    	this.metrics = metrics;
    }
}