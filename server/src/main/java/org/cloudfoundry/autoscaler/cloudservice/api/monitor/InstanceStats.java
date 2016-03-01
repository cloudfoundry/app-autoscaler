package org.cloudfoundry.autoscaler.cloudservice.api.monitor;

import java.util.ArrayList;


public class InstanceStats {
	
	public String appId;
	public String instanceID;
	public double numCores;
	public long   timeMsecs;
	public int    cpuPerc;
	public int    memMB;
	
	// list for customized metrics
	public ArrayList<MetricEntry> entryList = null;
	
	public InstanceStats() {
		this.entryList = new ArrayList<MetricEntry>();
	}
	
	public void addCustomizedMetric(MetricEntry entry) {
		this.entryList.add(entry);
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

	public double getNumCores() {
		return numCores;
	}

	public void setNumCores(double numCores) {
		this.numCores = numCores;
	}

	public long getTimeMsecs() {
		return timeMsecs;
	}

	public void setTimeMsecs(long timeMsecs) {
		this.timeMsecs = timeMsecs;
	}

	public int getCpuPerc() {
		return cpuPerc;
	}

	public void setCpuPerc(int cpuPerc) {
		this.cpuPerc = cpuPerc;
	}

	public int getMemMB() {
		return memMB;
	}

	public void setMemMB(int memMB) {
		this.memMB = memMB;
	}
	
}
