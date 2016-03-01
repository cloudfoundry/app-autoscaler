package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.BoundApp;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.ComplexKey;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;


@View(name = "by_serviceId", map = "function(doc) { if (doc.type=='BoundApp' && doc.serviceId) { emit([doc.serviceId], doc._id) } }")
public class BoundAppRepository_ByServiceId extends TypedCouchDbRepositorySupport<BoundApp> {
    public BoundAppRepository_ByServiceId(CouchDbConnector db) {
        super(BoundApp.class, db, "BoundApp_ByServiceId");
    }

    public List<BoundApp> findByServiceId(String serviceId) throws Exception {
        ComplexKey key = ComplexKey.of( serviceId);
        return queryView("by_serviceId", key);
    }

}
