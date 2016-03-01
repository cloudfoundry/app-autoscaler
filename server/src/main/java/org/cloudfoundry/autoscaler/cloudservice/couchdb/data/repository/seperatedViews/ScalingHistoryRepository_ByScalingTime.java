package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.ScalingHistory;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.ComplexKey;
import org.ektorp.CouchDbConnector;
import org.ektorp.ViewQuery;
import org.ektorp.support.View;

@View (name = "findByScalingTime", map = "function(doc) { if (doc.type == 'ScalingHistory' && doc.appId &&doc.startTime" +
		") emit( [doc.appId, doc.startTime], doc._id )}")
public class ScalingHistoryRepository_ByScalingTime extends TypedCouchDbRepositorySupport<ScalingHistory>{

	public ScalingHistoryRepository_ByScalingTime(CouchDbConnector db) {
		super(ScalingHistory.class, db, "ScalingHistory_ByScalingTime");
	}

	public List<ScalingHistory> findByScalingTime(String appId, long startTime, long endTime){
        ComplexKey startKey = ComplexKey.of( appId, endTime);
        ComplexKey endKey = ComplexKey.of( appId, startTime);
        ViewQuery q = createQuery("findByScalingTime").includeDocs(true).startKey(startKey).endKey(endKey).descending(true);
        

        List<ScalingHistory> returnvalue = null;
		String[] input = beforeConnection("QUERY", new String[]{"findByScalingTime", appId, String.valueOf(startTime), String.valueOf(endTime)});    	
    	try {
            returnvalue = db.queryView(q, ScalingHistory.class);
		} catch (Exception e){
			e.printStackTrace();
		}
    	afterConnection(input);

    	return returnvalue;
	}	

}
