package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.Application;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

@View(name = "by_appId", map = "function(doc) { if(doc.type=='Application' && doc.appId) {emit(doc.appId, doc._id)} }" )
public class ApplicationRepository_ByAppId extends TypedCouchDbRepositorySupport<Application>{
	public ApplicationRepository_ByAppId(CouchDbConnector db) {
		super(Application.class, db, "Application_ByAppId");
	}

	
	public Application findByAppId(String appId){
		if (appId == null)
			return null;
		List<Application> apps = queryView("by_appId", appId);
		if (apps == null || apps.size() == 0){
			return null;
		}
		return apps.get(0);
	}
	
	public List<Application> findDupliateByAppId(String appId){
		if (appId == null)
			return null;
		List<Application> apps = queryView("by_appId", appId);
		if (apps == null || apps.size() == 0){
			return null;
		}
		return apps;
	}
		
	
}
