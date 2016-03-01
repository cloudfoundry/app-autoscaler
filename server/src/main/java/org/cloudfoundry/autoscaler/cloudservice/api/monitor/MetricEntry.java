package org.cloudfoundry.autoscaler.cloudservice.api.monitor;

/**
 * For defining customized metrics
 * @author Shicong
 *
 */
public class MetricEntry {
	
	public String id = null;
	public String name = null;
	public double value = -1;
	
	public MetricEntry() {
		
	}
	
	public MetricEntry(String id, String name, double value) {
		this.id = id;
		this.name = name;
		this.value = value;
	}

}
