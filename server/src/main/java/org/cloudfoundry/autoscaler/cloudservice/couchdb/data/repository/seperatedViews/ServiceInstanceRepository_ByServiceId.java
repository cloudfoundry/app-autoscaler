package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.ServiceInstance;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

@View ( name = "by_serviceId", map = "function(doc) { if(doc.type=='ServiceInstance_inBroker' && doc.serviceId) {emit(doc.serviceId, doc._id)} }")
public class ServiceInstanceRepository_ByServiceId extends TypedCouchDbRepositorySupport<ServiceInstance> {

    public ServiceInstanceRepository_ByServiceId(CouchDbConnector db) {
    	super(ServiceInstance.class, db, "ServiceInstance_ByServiceId");  
    	initStandardDesignDocument();
    }
    
    public List<ServiceInstance> findByServiceId(String serviceId) {
    	return queryView("by_serviceId", serviceId); 
    }
    
    public List<ServiceInstance> findByAll() {
    	return queryView("by_serviceId"); 
    }

	public boolean updateServerURL (String serviceId, String serverURL){
		
		List<ServiceInstance> serviceInstances = this.findByServiceId(serviceId);
		if (serviceInstances != null) {
			for (ServiceInstance  serviceInstance : serviceInstances) {
				serviceInstance.setServerUrl(serverURL);
				update(serviceInstance);
			}
			return true;
		} else
			return false;
		
	}
    
    
}
