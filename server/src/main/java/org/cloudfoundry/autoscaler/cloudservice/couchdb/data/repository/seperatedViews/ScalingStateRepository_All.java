package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.AppAutoScaleState;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;


@View ( name = "byAll", map = "function(doc) { if (doc.type == 'AppAutoScaleState' ) emit( [doc.appId, doc.instanceCountState], doc._id )}")
public class ScalingStateRepository_All extends TypedCouchDbRepositorySupport<AppAutoScaleState>{

	public ScalingStateRepository_All(CouchDbConnector db) {
		super(AppAutoScaleState.class, db, "AppAutoScaleState_byAll");
	}

    public List<AppAutoScaleState> getAllRecords(){
    	return queryView("byAll");
    }

}
