package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.AppInstanceMetrics;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.ComplexKey;
import org.ektorp.CouchDbConnector;
import org.ektorp.ViewQuery;
import org.ektorp.support.View;


@View(name = "by_serviceId_before", map = "function(doc) { if (doc.type=='AppInstanceMetrics' && doc.serviceId && doc.timestamp) { emit([ doc.serviceId, doc.timestamp], doc._id) } }")
public class AppInstanceMetricsRepository_ByServiceId_Before extends TypedCouchDbRepositorySupport<AppInstanceMetrics> {

    public AppInstanceMetricsRepository_ByServiceId_Before(CouchDbConnector db) {
        super(AppInstanceMetrics.class, db, "AppInstanceMetrics_ByServiceId");
    }
    
     public List<AppInstanceMetrics> findByServiceIdBefore(String serviceId, long olderThan) throws Exception {
        ComplexKey startKey = ComplexKey.of( serviceId, 0);
        ComplexKey endKey = ComplexKey.of( serviceId, olderThan);
        ViewQuery q = createQuery("by_serviceId_before").includeDocs(true).startKey(startKey).endKey(endKey);
        
        
        List<AppInstanceMetrics> returnvalue = null;
  		String[] input = beforeConnection("QUERY", new String[]{"by_serviceId_before", serviceId, String.valueOf(0), String.valueOf(olderThan)});    	
      	try {
             returnvalue = db.queryView(q, AppInstanceMetrics.class);
  		} catch (Exception e){
  			e.printStackTrace();
  		}
      	afterConnection(input);
        
	    return returnvalue;
    }
    
}
