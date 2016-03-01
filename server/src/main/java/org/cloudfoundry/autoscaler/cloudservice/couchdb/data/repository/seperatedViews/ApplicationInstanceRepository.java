package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.ApplicationInstance;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.CouchDbRepositorySupport;
import org.ektorp.support.GenerateView;

public class ApplicationInstanceRepository extends CouchDbRepositorySupport<ApplicationInstance>{
	
    public ApplicationInstanceRepository(CouchDbConnector db) {
        super(ApplicationInstance.class, db, true);
        initStandardDesignDocument();
    }
    
    @GenerateView
    public List<ApplicationInstance> findByServiceId(String serviceId) {
        return queryView("by_serviceId", serviceId);
    }
    
    @GenerateView
    public List<ApplicationInstance> findByBindingId(String bindingId) {
        return queryView("by_bindingId", bindingId);
    }
}
