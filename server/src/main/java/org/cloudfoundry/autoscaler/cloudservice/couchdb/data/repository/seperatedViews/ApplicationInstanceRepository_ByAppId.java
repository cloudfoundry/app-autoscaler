package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.ApplicationInstance;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

@View ( name = "by_appId", map = "function(doc) { if(doc.type=='ApplicationInstance_inBroker' && doc.appId) {emit(doc.appId, doc._id)} }")
public class ApplicationInstanceRepository_ByAppId extends TypedCouchDbRepositorySupport<ApplicationInstance>{
	
    public ApplicationInstanceRepository_ByAppId(CouchDbConnector db) {
    	super(ApplicationInstance.class, db, "ApplicationInstance_ByAppId");
    	initStandardDesignDocument();
    }
    
    public List<ApplicationInstance> findByAppId(String appId) {
    	return  queryView("by_appId", appId);
    }

    public List<ApplicationInstance> findByAll() {
    	return queryView("by_appId"); 
    }
}