package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.Application;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

@View ( name = "byAll", map = "function(doc) { if (doc.type == 'Application' ) emit([doc.appId, doc.serviceId, doc.bindingId, doc.policyId, doc.state], doc._id )}")
public class ApplicationRepository_All extends TypedCouchDbRepositorySupport<Application>{
	public ApplicationRepository_All(CouchDbConnector db) {
		super(Application.class, db, "Application_byAll");
	}

	
    public List<Application> getAllRecords(){
    	return queryView("byAll");
    }

	
}
