package org.cloudfoundry.autoscaler.servicebroker.data.entity;

import org.cloudfoundry.autoscaler.servicebroker.Constants;
import org.ektorp.support.TypeDiscriminator;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

@JsonIgnoreProperties(ignoreUnknown = true)
@TypeDiscriminator ("doc.type=='ServiceInstance_inBroker'")
public class ServiceInstance extends TypedCouchDbDocument  {
    
	/**
	 * 
	 */
	private static final long serialVersionUID = 1L;
	private String serviceId; //unique id
    private String serverUrl;
    private String orgId;
	private String spaceId;
	private long timestamp;
	
	public ServiceInstance() {
		this.type = Constants.COUCHDOCUMENT_TYPE_SERVICEINSTANCE;
    }

	public String getServiceId() {
		return serviceId;
	}

	public void setServiceId(String serviceId) {
		this.serviceId = serviceId;
	}

	public String getServerUrl() {
		return serverUrl;
	}

	public void setServerUrl(String serverUrl) {
		this.serverUrl = serverUrl;
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

	public long getTimestamp() {
		return timestamp;
	}

	public void setTimestamp(long timestamp) {
		this.timestamp = timestamp;
	}



	
}
