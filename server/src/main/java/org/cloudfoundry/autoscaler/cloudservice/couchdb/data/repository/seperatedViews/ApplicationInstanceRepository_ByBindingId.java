package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.ApplicationInstance;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

@View ( name = "by_bindingId", map = "function(doc) { if(doc.type=='ApplicationInstance_inBroker' && doc.bindingId) {emit(doc.bindingId, doc._id)} }")
public class ApplicationInstanceRepository_ByBindingId extends TypedCouchDbRepositorySupport<ApplicationInstance>{
	
    public ApplicationInstanceRepository_ByBindingId(CouchDbConnector db) {
    	super(ApplicationInstance.class, db, "ApplicationInstance_ByBindingId");
    	initStandardDesignDocument();
    }
    
    public List<ApplicationInstance> findByBindingId(String bindingId) {
    	return queryView("by_bindingId", bindingId);
    }
    	


}
