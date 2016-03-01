package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.ApplicationInstance;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

@View ( name = "by_serviceId", map = "function(doc) { if(doc.type=='ApplicationInstance_inBroker' && doc.serviceId) {emit(doc.serviceId, doc._id)} }")
public class ApplicationInstanceRepository_ByServiceId extends TypedCouchDbRepositorySupport<ApplicationInstance>{
	
    public ApplicationInstanceRepository_ByServiceId(CouchDbConnector db) {
    	super(ApplicationInstance.class, db, "ApplicationInstance_ByServiceId");
    	initStandardDesignDocument();
    }
    
    public List<ApplicationInstance> findByServiceId(String serviceId) {
        return queryView("by_serviceId", serviceId);
    }


}
