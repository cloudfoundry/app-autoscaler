package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.AppInstanceMetrics;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;


@View ( name = "byAll", map = "function(doc) { if (doc.type == 'AppInstanceMetrics' ) emit([doc.appId, doc.appType, doc.timestamp], doc._id)}")
public class AppInstanceMetricsRepository_All extends TypedCouchDbRepositorySupport<AppInstanceMetrics> {

    public AppInstanceMetricsRepository_All(CouchDbConnector db) {
        super(AppInstanceMetrics.class, db, "AppInstanceMetrics_byAll");
    }
 
    public List<AppInstanceMetrics> getAllRecords(){
    	return queryView("byAll");
    }

}
