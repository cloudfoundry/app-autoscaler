package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.AppInstanceMetrics;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.ComplexKey;
import org.ektorp.CouchDbConnector;
import org.ektorp.ViewQuery;
import org.ektorp.support.View;


@View(name = "by_appId_between", map = "function(doc) { if (doc.type=='AppInstanceMetrics' && doc.appId && doc.timestamp) { emit([doc.appId, doc.timestamp], doc._id) } }")
public class AppInstanceMetricsRepository_ByAppIdBetween extends TypedCouchDbRepositorySupport<AppInstanceMetrics> {

    public AppInstanceMetricsRepository_ByAppIdBetween(CouchDbConnector db) {
        super(AppInstanceMetrics.class, db, "AppInstanceMetrics_ByAppIdBetween");
    }
    
    
    public List<AppInstanceMetrics> findByAppIdBetween(String appId, long startTimestamp, long endTimestamp) throws Exception {

    	
    	ComplexKey startKey = ComplexKey.of( appId, startTimestamp);
        ComplexKey endKey = ComplexKey.of( appId, endTimestamp);
        ViewQuery q = createQuery("by_appId_between").includeDocs(true).startKey(startKey).endKey(endKey);
        

        List<AppInstanceMetrics> returnvalue = null;
 		String[] input = beforeConnection("QUERY", new String[]{"by_appId_between", appId, String.valueOf(startTimestamp), String.valueOf(endTimestamp)});    	
     	try {
            returnvalue = db.queryView(q, AppInstanceMetrics.class);
 		} catch (Exception e){
 			e.printStackTrace();
 		}
     	afterConnection(input);
     
    	return returnvalue;
    }
    
    
}
