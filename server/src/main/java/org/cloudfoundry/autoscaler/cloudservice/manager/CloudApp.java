package org.cloudfoundry.autoscaler.cloudservice.manager;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;


@JsonIgnoreProperties(ignoreUnknown = true)
public class CloudApp
{

	private String                      appName;
	private String                      buildpack;
	private int instances; //All instances	
	
	public CloudApp()
	{
	}
	
	public CloudApp(String name)
	{
		appName        = name;
	}
	
	public String getAppName() {
		return appName;
	}

	public void setAppName(String name) {
		this.appName = name;
	}

	public int getInstances() {
		return instances;
	}

	public void setInstances(int instances) {
		this.instances = instances;
	}

	public String getBuildpack() {
		return buildpack;
	}

	public void setBuildpack(String buildpack) {
		this.buildpack = buildpack;
	}
	
}
