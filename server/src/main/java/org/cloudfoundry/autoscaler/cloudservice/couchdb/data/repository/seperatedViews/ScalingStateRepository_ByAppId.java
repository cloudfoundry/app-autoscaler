package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.AppAutoScaleState;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;


@View ( name = "by_appId", map = "function(doc) { if(doc.type=='AppAutoScaleState' && doc.appId) {emit(doc.appId, doc._id)} }")
public class ScalingStateRepository_ByAppId extends TypedCouchDbRepositorySupport<AppAutoScaleState>{

	public ScalingStateRepository_ByAppId(CouchDbConnector db) {
		super(AppAutoScaleState.class, db, "AppAutoScaleState_ByAppId");
	}
	
	public AppAutoScaleState findByAppId(String appId){
		if (appId == null)
			return null;
		
		List<AppAutoScaleState> states = queryView("by_appId", appId);
		if (states == null || states.size() == 0){
			return null;
		}
		return states.get(0);		
	}


}
