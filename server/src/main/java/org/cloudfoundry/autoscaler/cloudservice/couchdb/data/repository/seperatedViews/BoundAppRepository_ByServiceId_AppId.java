package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.BoundApp;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.ComplexKey;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;


@View(name = "by_serviceId_appId", map = "function(doc) { if (doc.type=='BoundApp' && doc.serviceId && doc.appId) { emit([doc.serviceId, doc.appId], doc._id) } }")
public class BoundAppRepository_ByServiceId_AppId extends TypedCouchDbRepositorySupport<BoundApp> {
    public BoundAppRepository_ByServiceId_AppId(CouchDbConnector db) {
        super(BoundApp.class, db, "BoundApp_ByServiceId_AppId");
    }

    public List<BoundApp> findByServiceIdAndAppId(String serviceId, String appId) throws Exception {
        ComplexKey key = ComplexKey.of( serviceId, appId);
        return queryView("by_serviceId_appId", key);
    }    

   
}
