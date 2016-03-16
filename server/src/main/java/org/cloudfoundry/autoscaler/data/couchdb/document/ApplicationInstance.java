package org.cloudfoundry.autoscaler.data.couchdb.document;

import org.ektorp.support.TypeDiscriminator;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

@JsonIgnoreProperties(ignoreUnknown = true)
@TypeDiscriminator ("doc.type=='ApplicationInstance_inBroker'")
public class ApplicationInstance extends TypedCouchDbDocument {
	
	/**
	 * 
	 */
	private static final long serialVersionUID = 1L;

	public static enum ApplicationInstanceState{
		Creating, Registering, Bound, Unregistering, Removing
	}
	
	public static final short ApplicationInstance_State_Creating = 1;
	public static final short ApplicationInstance_State_Registering = 2;
	public static final short ApplicationInstance_State_Bound = 3;
	public static final short ApplicationInstance_State_Unregistering = 4;
	public static final short ApplicationInstance_State_Removing = 5;

	
	private String bindingId;
    private String serviceId;
    private String appId;
    private ApplicationInstanceState state;
    
    public ApplicationInstance() {
    	this.type = "ApplicationInstance_inBroker";
    }
   
	public String getBindingId() {
		return bindingId;
	}

	public void setBindingId(String bindingId) {
		this.bindingId = bindingId;
	}
	
	public String getServiceId() {
		return serviceId;
	}

	public void setServiceId(String serviceId) {
		this.serviceId = serviceId;
	}

	public String getAppId() {
		return appId;
	}

	public void setAppId(String appId) {
		this.appId = appId;
	}

	public ApplicationInstanceState getState() {
		return state;
	}

	public void setState(ApplicationInstanceState state) {
		this.state = state;
	}
	
	
	
}
