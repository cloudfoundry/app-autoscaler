package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.Application;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

@View(name = "by_serviceId", map = "function(doc) { if(doc.type=='Application' && doc.serviceId) {emit(doc.serviceId, doc._id)} }" )
public class ApplicationRepository_ByServiceId extends TypedCouchDbRepositorySupport<Application>{
	public ApplicationRepository_ByServiceId(CouchDbConnector db) {
		super(Application.class, db, "Application_ByServiceId");
	}

	public List<Application> findByServiceId(String serviceId){
		return queryView("by_serviceId", serviceId);
	}

	
}
