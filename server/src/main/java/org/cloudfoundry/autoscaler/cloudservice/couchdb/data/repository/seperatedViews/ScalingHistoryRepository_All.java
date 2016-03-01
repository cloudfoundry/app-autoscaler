package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.ScalingHistory;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

@View ( name = "byAll", map = "function(doc) { if (doc.type == 'ScalingHistory' ) emit( [doc.appId, doc.status], doc._id )}")
public class ScalingHistoryRepository_All extends TypedCouchDbRepositorySupport<ScalingHistory>{

	public ScalingHistoryRepository_All(CouchDbConnector db) {
		super(ScalingHistory.class, db, "ScalingHistory_byAll");
	}

    public List<ScalingHistory> getAllRecords(){
    	return queryView("byAll");
    }


}
