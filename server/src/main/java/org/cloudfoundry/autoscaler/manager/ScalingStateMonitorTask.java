package org.cloudfoundry.autoscaler.manager;

public class ScalingStateMonitorTask {
	private String appId;
	private String org;
	private String space;
	private String scaclingActionId;
	private int targetInstanceCount;
	
	public ScalingStateMonitorTask(String appId,
			int targetInstanceCount, String actionId) {
		super();
		this.appId = appId;
		this.targetInstanceCount = targetInstanceCount;
		this.scaclingActionId = actionId;
	}
	public String getAppId() {
		return appId;
	}
	public void setAppId(String appName) {
		this.appId = appName;
	}
	public String getOrg() {
		return org;
	}
	public void setOrg(String org) {
		this.org = org;
	}
	public String getSpace() {
		return space;
	}
	public void setSpace(String space) {
		this.space = space;
	}
	public int getTargetInstanceCount() {
		return targetInstanceCount;
	}
	public void setTargetInstanceCount(int targetInstanceCount) {
		this.targetInstanceCount = targetInstanceCount;
	}
	public String getScaclingActionId() {
		return scaclingActionId;
	}
	public void setScaclingActionId(String scaclingActionId) {
		this.scaclingActionId = scaclingActionId;
	}

}
