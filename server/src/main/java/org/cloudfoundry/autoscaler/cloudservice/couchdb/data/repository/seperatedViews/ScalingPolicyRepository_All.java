package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.AutoScalerPolicy;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;


@View ( name = "byAll", map = "function(doc) { if (doc.type == 'AutoScalerPolicy' ) emit( [doc.policyId, doc.instanceMinCount, doc.instanceMaxCount], doc._id )}")
public class ScalingPolicyRepository_All extends TypedCouchDbRepositorySupport<AutoScalerPolicy> 
{

	public ScalingPolicyRepository_All(CouchDbConnector db) {
		super(AutoScalerPolicy.class, db, "AutoScalerPolicy_byAll");
	}

    public List<AutoScalerPolicy> getAllRecords(){
    	return queryView("byAll");
    }

}
