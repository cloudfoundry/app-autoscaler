package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.ServiceInstance;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

@View ( name = "without_serviceId", map = "function(doc) { if(doc.type=='ServiceInstance_inBroker' && !doc.serviceId) {emit(doc._id)} }")
public class ServiceInstanceRepository_WithoutServiceId extends TypedCouchDbRepositorySupport<ServiceInstance> {

    public ServiceInstanceRepository_WithoutServiceId(CouchDbConnector db) {
    	super(ServiceInstance.class, db, "ServiceInstance_WithoutServiceId");  
    	initStandardDesignDocument();
    }
    
    public List<ServiceInstance> findByServiceId(String serviceId) {
    	return queryView("without_serviceId", serviceId); 
    }
    
}
