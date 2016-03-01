package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.TriggerRecord;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;



@View ( name = "byAll", map = "function(doc) { if (doc.type == 'TriggerRecord' ) emit( [doc.appId, doc.serverName], doc._id )}")
public class TriggerRecordRepository_All extends TypedCouchDbRepositorySupport<TriggerRecord> {

    public TriggerRecordRepository_All(CouchDbConnector db) {
        super(TriggerRecord.class, db, "TriggerRecord_byAll");
    }

    public List<TriggerRecord> getAllRecords(){
    	return queryView("byAll");
    }

}
