package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.BoundApp;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.ComplexKey;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;


@View(name = "by_appId", map = "function(doc) { if (doc.type=='BoundApp' && doc.appId) { emit([doc.appId], doc._id) } }")
public class BoundAppRepository_ByAppId extends TypedCouchDbRepositorySupport<BoundApp> {
    public BoundAppRepository_ByAppId(CouchDbConnector db) {
        super(BoundApp.class, db, "BoundApp_ByAppId");
    }

    public BoundApp findByAppId(String appId) throws Exception {
        ComplexKey key = ComplexKey.of( appId);
		List<BoundApp> apps = queryView("by_appId", ComplexKey.of(appId));
		if (apps == null || apps.size() == 0){
			return null;
		}
        return apps.get(0);
    }
    
    
	public List<BoundApp> findDuplicateByAppId(String appId){
		if (appId == null)
			return null;
		List<BoundApp> apps = queryView("by_appId", ComplexKey.of(appId));
		if (apps == null || apps.size() == 0){
			return null;
		}
		return apps;
	}
}
