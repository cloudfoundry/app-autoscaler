package org.cloudfoundry.autoscaler.manager;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

/**
 * This class is used to represent the request content for addApp rest api
 * 
 * 
 * 
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class NewAppRequestEntity {
	private String appId; // app GUID
	private String serviceId; // scaling service instance ID
	private String bindingId; // binding Id
	private String orgId;
	private String spaceId;

	public String getAppId() {
		return appId;
	}

	public void setAppId(String appId) {
		this.appId = appId;
	}

	public String getServiceId() {
		return serviceId;
	}

	public void setServiceId(String serviceId) {
		this.serviceId = serviceId;
	}

	public String getBindingId() {
		return bindingId;
	}

	public void setBindingId(String bindingId) {
		this.bindingId = bindingId;
	}

	public String getOrgId() {
		return orgId;
	}

	public void setOrgId(String orgId) {
		this.orgId = orgId;
	}

	public String getSpaceId() {
		return spaceId;
	}

	public void setSpaceId(String spaceId) {
		this.spaceId = spaceId;
	}

}
