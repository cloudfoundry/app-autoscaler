package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.ApplicationInstance;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

@View ( name = "without_appId", map = "function(doc) { if(doc.type=='ApplicationInstance_inBroker' && !doc.appId) {emit(doc._id)} }")
public class ApplicationInstanceRepository_WithoutAppId extends TypedCouchDbRepositorySupport<ApplicationInstance>{
	
    public ApplicationInstanceRepository_WithoutAppId(CouchDbConnector db) {
    	super(ApplicationInstance.class, db, "ApplicationInstance_WithoutAppId");
    	initStandardDesignDocument();
    }
    
    public List<ApplicationInstance> findByAppId(String appId) {
    	return  queryView("without_appId", appId);
    }

}
