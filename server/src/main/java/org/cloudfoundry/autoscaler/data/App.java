package org.cloudfoundry.autoscaler.data;


import com.fasterxml.jackson.annotation.JsonIgnoreProperties;


@JsonIgnoreProperties(ignoreUnknown = true)
public class App
{

	private String appName;
	private String policyId;
	private String handleId;

	
	public App()
	{
	}
	
	public App(String name, String policy)
	{
		appName    = name;
		policyId = policy;
	}

	public App(String name, String policy, String handleId)
	{
		appName    = name;
		policyId = policy;
		this.handleId = handleId;
	}
	public String getAppName() {
		return appName;
	}
	public void setAppName(String appName) {
		this.appName = appName;
	}

	public String getPolicyId() {
		return policyId;
	}

	public void setPolicyId(String policyId) {
		this.policyId = policyId;
	}

	public String getHandleId() {
		return handleId;
	}

	public void setHandleId(String handleId) {
		this.handleId = handleId;
	}
	
}
