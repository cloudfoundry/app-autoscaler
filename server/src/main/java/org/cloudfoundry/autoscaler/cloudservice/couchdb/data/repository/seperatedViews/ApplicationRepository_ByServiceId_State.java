package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.Application;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

@View(name = "by_serviceId_state", map = "function(doc) { if (doc.type == 'Application' &&  doc.state != 'unbond' && doc.serviceId ) emit( doc.serviceId, doc._id )}" )
public class ApplicationRepository_ByServiceId_State extends TypedCouchDbRepositorySupport<Application>{
	public ApplicationRepository_ByServiceId_State(CouchDbConnector db) {
		super(Application.class, db, "Application_ByServiceId_State");
	}

	public List<Application> findByServiceIdAndState(String serviceId){
		return queryView("by_serviceId_state", serviceId);
	}


	
}
