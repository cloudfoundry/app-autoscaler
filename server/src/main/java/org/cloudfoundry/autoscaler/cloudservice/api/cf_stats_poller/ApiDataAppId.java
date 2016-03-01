package org.cloudfoundry.autoscaler.cloudservice.api.cf_stats_poller;

public class ApiDataAppId
{

	private String appId;
	
	public ApiDataAppId() {
	}
	public ApiDataAppId(String id) {
		appId = id;
	}
	
	public String getAppId() {
		return appId;
	}
	public void setAppId(String appId) {
		this.appId = appId;
	}
	
}
