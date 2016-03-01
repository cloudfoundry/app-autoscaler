package org.cloudfoundry.autoscaler.servicebroker.data.entity;

import org.cloudfoundry.autoscaler.servicebroker.Constants;
import org.ektorp.support.TypeDiscriminator;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

@JsonIgnoreProperties(ignoreUnknown = true)
@TypeDiscriminator ("doc.type=='ApplicationInstance_inBroker'")
public class ApplicationInstance extends TypedCouchDbDocument {
	
	/**
	 * 
	 */
	private static final long serialVersionUID = 1L;
	private String bindingId; //unique id
    private String serviceId;
    private String appId;
    private long timestamp;
    
    public ApplicationInstance() {
    	this.type = Constants.COUCHDOCUMENT_TYPE_APPLICATIONINSTANCE;
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

	public long getTimestamp() {
		return timestamp;
	}

	public void setTimestamp(long timestamp) {
		this.timestamp = timestamp;
	}

	
}
