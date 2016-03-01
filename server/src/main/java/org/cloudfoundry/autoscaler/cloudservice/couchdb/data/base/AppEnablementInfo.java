package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.base;

public class AppEnablementInfo {
	private String appId;
    private String bindingId;
	private String orgId;
	private String spaceId;
    
    public String getAppId() {
		return appId;
	}
	public void setAppId(String appId) {
		this.appId = appId;
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
