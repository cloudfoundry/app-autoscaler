package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.Application;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

@View(name = "by_bindingId", map = "function(doc) { if(doc.type=='Application' && doc.bindingId) {emit(doc.bindingId, doc._id)} }" )
public class ApplicationRepository_ByBindingId extends TypedCouchDbRepositorySupport<Application>{
	public ApplicationRepository_ByBindingId(CouchDbConnector db) {
		super(Application.class, db, "Application_ByBindingId");
	}


	public Application findByBindingId(String bindingId){
		if (bindingId == null)
			return null;
		List<Application> apps = queryView("by_bindingId", bindingId);
		if (apps == null || apps.size() == 0){
			return null;
		}
		return apps.get(0);
			
	}

	
	
}
