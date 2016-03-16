package org.cloudfoundry.autoscaler.data.couchdb.document;

import org.ektorp.support.TypeDiscriminator;

@TypeDiscriminator ("doc.type=='ApplicationInstance_inBroker'")
public class ApplicationInstanceInBroker extends TypedCouchDbDocument {
	
	/**
	 * 
	 */
	private static final long serialVersionUID = 1L;
	private String bindingId;
    private String serviceId;
    
    public ApplicationInstanceInBroker() {
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
}
