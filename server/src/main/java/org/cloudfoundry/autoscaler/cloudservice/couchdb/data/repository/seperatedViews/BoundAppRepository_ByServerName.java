package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.BoundApp;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.ComplexKey;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;


@View(name = "by_servername", map = "function(doc) { if (doc.type=='BoundApp' && doc.serverName) { emit([doc.serverName], doc._id) } }")
public class BoundAppRepository_ByServerName extends TypedCouchDbRepositorySupport<BoundApp> {
    public BoundAppRepository_ByServerName(CouchDbConnector db) {
        super(BoundApp.class, db, "BoundApp_ByServerName");
    }

    public List<BoundApp> getAllBoundApps(String serverName) {
        ComplexKey key = ComplexKey.of( serverName);
        return queryView("by_servername", key);
    }    
	
   
}
