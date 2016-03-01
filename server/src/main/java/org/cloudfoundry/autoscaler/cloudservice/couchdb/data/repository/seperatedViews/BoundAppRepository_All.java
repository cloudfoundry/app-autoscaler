package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.BoundApp;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;


@View ( name = "byAll", map = "function(doc) { if (doc.type == 'BoundApp' ) emit( [doc.appId, doc.serverName], doc._id )}")
public class BoundAppRepository_All extends TypedCouchDbRepositorySupport<BoundApp> {
    public BoundAppRepository_All(CouchDbConnector db) {
        super(BoundApp.class, db, "BoundApp_byAll");
    }

    public List<BoundApp> getAllRecords(){
    	return queryView("byAll");
    }

}
