package org.cloudfoundry.autoscaler.cloudservice.api.monitor;

import java.util.List;
import java.util.Vector;

public class InstanceUpdate {
	
	public String appId;			//id of the monitored application
	public String instanceID;		//id of the instance
	public List<String> metricIdList = new Vector<String>();	//ordered list of metric id
	public List<Double> valueList    = new Vector<Double>();	//ordered list of corresponding metric values
	public long 	timestamp;		//local timestamp for the collected metric values
	
	public InstanceUpdate() {
		
	}
	
	public void addMetricValue(String metricId, double value) {
		this.metricIdList.add(metricId);
		this.valueList.add(value);
	}
	
	public String getAppId() {
		return appId;
	}
	
	public void setAppId(String appId) {
		this.appId = appId;
	}
	
	public String getInstanceID() {
		return instanceID;
	}
	
	public void setInstanceID(String instanceID) {
		this.instanceID = instanceID;
	}
	
	public List<String> getMetricIdList() {
		return metricIdList;
	}
	
	public void setMetricIdList(List<String> metricIdList) {
		this.metricIdList = metricIdList;
	}
	
	public List<Double> getValueList() {
		return valueList;
	}
	
	public void setValueList(List<Double> valueList) {
		this.valueList = valueList;
	}
	
	public long getTimestamp() {
		return timestamp;
	}
	
	public void setTimestamp(long timestamp) {
		this.timestamp = timestamp;
	}

	

}
