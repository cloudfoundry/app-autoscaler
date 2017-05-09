package org.cloudfoundry.autoscaler.scheduler.util;

import org.cloudfoundry.autoscaler.scheduler.entity.PolicyJsonEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationSchedules;
import org.cloudfoundry.autoscaler.scheduler.rest.model.Schedules;

public class PolicyJsonEntityBuilder {
	
	private PolicyJsonEntity policyJson;
	
	public PolicyJsonEntityBuilder(){
		this.policyJson = new PolicyJsonEntity();
	}
	
	public PolicyJsonEntityBuilder(String appId, String guid){
		this();
		this.policyJson.setAppId(appId);
		this.policyJson.setGuid(guid);
	}
	
	public PolicyJsonEntityBuilder(String appId, String guid, Schedules schedules){
		this();
		this.policyJson.setAppId(appId);
		this.policyJson.setGuid(guid);
		ApplicationSchedules applicationSchedules = new ApplicationSchedules();
		applicationSchedules.setInstanceMaxCount(1);
		applicationSchedules.setInstanceMaxCount(5);
		applicationSchedules.setSchedules(schedules);
		this.policyJson.setSchedules(applicationSchedules);
	}
	
	public PolicyJsonEntity build(){
		return this.policyJson;
	}

}
