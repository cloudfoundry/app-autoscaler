package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.AutoScalerPolicy;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.cloudfoundry.autoscaler.exceptions.PolicyNotFoundException;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;


@View ( name = "by_policyId", map = "function(doc) { if(doc.type=='AutoScalerPolicy' && doc.policyId) {emit(doc.policyId, doc._id)}}")
public class ScalingPolicyRepository_ByPolicyId extends TypedCouchDbRepositorySupport<AutoScalerPolicy> 
{

	public ScalingPolicyRepository_ByPolicyId(CouchDbConnector db) {
		super(AutoScalerPolicy.class, db, "AutoScalerPolicy_ByPolicyId");
	}

	public AutoScalerPolicy findByPolicyId(String policyId) throws PolicyNotFoundException{
		if (policyId == null ) 
			return null;
		
		List<AutoScalerPolicy> policies = queryView("by_policyId", policyId);
		if (policies == null || policies.size() == 0){
			throw new PolicyNotFoundException(policyId);
		}
		return policies.get(0);
	}	

	
}
