package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.TriggerRecord;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.ComplexKey;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;



@View(name = "by_servername", map = "function(doc) { if (doc.type=='TriggerRecord' && doc.serverName) { emit([doc.serverName], doc._id) } }")
public class TriggerRecordRepository_ByServerName extends TypedCouchDbRepositorySupport<TriggerRecord> {

    public TriggerRecordRepository_ByServerName(CouchDbConnector db) {
        super(TriggerRecord.class, db, "TriggerRecord_ByServerName");
    }

    public List<TriggerRecord> findByServerName(String serverName){
        ComplexKey key = ComplexKey.of( serverName);
        return queryView("by_servername", key);    	
    }
    
}
