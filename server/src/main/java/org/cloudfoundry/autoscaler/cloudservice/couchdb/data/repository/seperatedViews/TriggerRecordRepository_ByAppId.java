package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.TriggerRecord;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.ComplexKey;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;



@View(name = "by_appId", map = "function(doc) { if (doc.type=='TriggerRecord' && doc.appId) { emit([ doc.appId], doc._id) } }")
public class TriggerRecordRepository_ByAppId extends TypedCouchDbRepositorySupport<TriggerRecord> {

    public TriggerRecordRepository_ByAppId(CouchDbConnector db) {
        super(TriggerRecord.class, db, "TriggerRecord_ByAppId");
    }

    
    public List<TriggerRecord> findByAppId(String appId) throws Exception {
        ComplexKey key = ComplexKey.of(appId);
        return queryView("by_appId", key);
    }
}
