package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.AppInstanceMetrics;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.ComplexKey;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;


@View(name = "by_appId", map = "function(doc) { if (doc.type=='AppInstanceMetrics' && doc.appId) { emit([doc.appId], doc._id) } }")
public class AppInstanceMetricsRepository_ByAppId extends TypedCouchDbRepositorySupport<AppInstanceMetrics> {

    public AppInstanceMetricsRepository_ByAppId(CouchDbConnector db) {
        super(AppInstanceMetrics.class, db, "AppInstanceMetrics_ByAppId");
    }
    
    
    public List<AppInstanceMetrics> findByAppId(String appId) {
        ComplexKey key = ComplexKey.of( appId);
        return queryView("by_appId", key);
    }

    
}
