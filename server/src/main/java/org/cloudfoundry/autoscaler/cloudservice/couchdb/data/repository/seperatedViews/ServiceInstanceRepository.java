package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.ServiceInstance;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.CouchDbRepositorySupport;
import org.ektorp.support.GenerateView;

public class ServiceInstanceRepository extends CouchDbRepositorySupport<ServiceInstance> {

    public ServiceInstanceRepository(CouchDbConnector db) {
        super(ServiceInstance.class, db, true);
        initStandardDesignDocument();
    }
    
    @GenerateView
    public List<ServiceInstance> findByServerUrl(String serverUrl) {
        return queryView("by_serverUrl", serverUrl);
    }
}
