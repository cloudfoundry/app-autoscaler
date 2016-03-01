package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document;

import org.ektorp.support.TypeDiscriminator;

@TypeDiscriminator ("doc.type=='ServiceInstance_inBroker'")
public class ServiceInstanceInBroker extends TypedCouchDbDocument {
    
    private String serverUrl;
    private String orgId;
	private String spaceId;

	public ServiceInstanceInBroker() {
		this.type = "ServiceInstance_inBroker";
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

}
