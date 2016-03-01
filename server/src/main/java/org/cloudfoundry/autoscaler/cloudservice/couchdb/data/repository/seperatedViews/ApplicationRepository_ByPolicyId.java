package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.Application;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

@View(name = "by_policyId", map = "function(doc) { if(doc.type=='Application' && doc.policyId) {emit(doc.policyId, doc._id)} }" )
public class ApplicationRepository_ByPolicyId extends TypedCouchDbRepositorySupport<Application>{
	public ApplicationRepository_ByPolicyId(CouchDbConnector db) {
		super(Application.class, db, "Application_ByPolicyId");
	}

	
	public List<Application> findByPolicyId(String policyId){
		return queryView("by_policyId", policyId);
	}

	
	
}
