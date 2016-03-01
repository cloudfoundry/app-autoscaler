package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.ServiceInstance;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

@View ( name = "by_serverUrl", map = "function(doc) { if(doc.type=='ServiceInstance_inBroker' && doc.serverUrl) {emit(doc.serverUrl, doc._id)} }")
public class ServiceInstanceRepository_ByServerURL extends TypedCouchDbRepositorySupport<ServiceInstance> {

    public ServiceInstanceRepository_ByServerURL(CouchDbConnector db) {
    	super(ServiceInstance.class, db, "ServiceInstance_ByServerURL"); 
    	initStandardDesignDocument();
    }
    
    public List<ServiceInstance> findByServerUrl(String serverUrl) {
        return queryView("by_serverUrl", serverUrl);
    }
    
}
