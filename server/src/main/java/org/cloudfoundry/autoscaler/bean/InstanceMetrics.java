package org.cloudfoundry.autoscaler.bean;

import java.io.Serializable;
import java.util.LinkedList;
import java.util.List;

import com.fasterxml.jackson.annotation.JsonIgnore;
import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

@JsonIgnoreProperties(ignoreUnknown = true)
public class InstanceMetrics implements Serializable, Cloneable {
    /**
	 * 
	 */
	private static final long serialVersionUID = 1L;
	private long timestamp;
    private int instanceIndex;
    private String instanceId;
    private List<Metric> metrics = new LinkedList<Metric>();

    @JsonIgnore
    private boolean  stored;

    public InstanceMetrics() {
    	this.stored = false;
    }

    public int getInstanceIndex() {
        return instanceIndex;
    }

    public void setInstanceIndex(int instanceIndex) {
        this.instanceIndex = instanceIndex;
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

    public List<Metric> getMetrics() {
        return metrics;
    }

    public void setMetrics(List<Metric> metrics) {
        this.metrics = metrics;
    }
    

	public boolean isStored() {
		return stored;
	}

	public void setStored(boolean stored) {
		this.stored = stored;
	}

	public InstanceMetrics merge(InstanceMetrics instMetrics) {
		//As the merge operation will change the original InstanceMetrics, clone it before merge.
		InstanceMetrics cloned = this.clone();
		cloned.metrics.addAll(instMetrics.getMetrics());
		return cloned;
    }

    public void addMetrics(List<Metric> metrics) {
        this.metrics.addAll(metrics);
    }

    public void addMetric(Metric metric) {
        this.metrics.add(metric);
    }

    @SuppressWarnings({ "rawtypes", "unchecked" })
    public InstanceMetrics clone() {
        try {
            InstanceMetrics cloned = (InstanceMetrics) super.clone();
            List clonedMetrics = new LinkedList<Metric>();
            for ( Metric metric : this.metrics ){
            	clonedMetrics.add(metric.clone());
            }
            
            cloned.setMetrics(clonedMetrics);
            return cloned;
        } catch (CloneNotSupportedException e) {
            throw new RuntimeException(e);
        }
    }
}
