package org.cloudfoundry.autoscaler.data.couchdb.document;

import org.ektorp.support.TypeDiscriminator;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

@JsonIgnoreProperties(ignoreUnknown = true)
@TypeDiscriminator ("doc.type=='Application'")
public class Application extends TypedCouchDbDocument{
	/**
	 * 
	 */
	private static final long serialVersionUID = 2241356985996248690L;
	private String appId; // application ID
	private String serviceId; // service Instance ID
	private String bindingId; // service binding ID
	private String policyId;
	private String policyState;// the policy is enabled or disabled
	private String appType; //app type can be java, node or ruby
	private String orgId;
	private String spaceId;
	private long bindTime;
	private String state; //state is disabled or enabled

	
	public Application(){
		super();
	}

	public Application(String appId, String serviceId, String bindingId, String orgId
			,String spaceId) {
		super();
		this.appId = appId;
		this.serviceId = serviceId;
		this.bindingId = bindingId;
		this.orgId = orgId;
		this.spaceId = spaceId;
	}
	
	
	
	public Application(String appId, String serviceId, String bindingId,
			String policyId, String policyState, String appType, String orgId,
			String spaceId, long bindTime, String state) {
		super();
		this.appId = appId;
		this.serviceId = serviceId;
		this.bindingId = bindingId;
		this.policyId = policyId;
		this.policyState = policyState;
		this.appType = appType;
		this.orgId = orgId;
		this.spaceId = spaceId;
		this.bindTime = bindTime;
		this.state = state;
	}
	
	
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
	public String getPolicyId() {
		return policyId;
	}
	public void setPolicyId(String policyId) {
		this.policyId = policyId;
	}
	public String getPolicyState() {
		return policyState;
	}
	public void setPolicyState(String policyState) {
		this.policyState = policyState;
	}
	public String getAppType() {
		return appType;
	}
	public void setAppType(String appType) {
		this.appType = appType;
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
	public long getBindTime() {
		return bindTime;
	}
	public void setBindTime(long bindTime) {
		this.bindTime = bindTime;
	}
	public String getState() {
		return state;
	}
	public void setState(String state) {
		this.state = state;
	}
	
}
