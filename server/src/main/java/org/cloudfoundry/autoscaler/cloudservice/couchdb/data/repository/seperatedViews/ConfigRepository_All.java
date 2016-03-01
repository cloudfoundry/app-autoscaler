package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.ServiceConfig;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;


@View ( name = "byAll", map = "function(doc) { if (doc.type == 'ServiceConfig' ) emit( [doc.serviceId], doc._id )}")
public class ConfigRepository_All extends TypedCouchDbRepositorySupport<ServiceConfig> {

    public ConfigRepository_All(CouchDbConnector db) {
        super(ServiceConfig.class, db, "ServiceConfig_byAll");
    }

    public List<ServiceConfig> getAllRecords(){
    	return queryView("byAll");
    }

}
