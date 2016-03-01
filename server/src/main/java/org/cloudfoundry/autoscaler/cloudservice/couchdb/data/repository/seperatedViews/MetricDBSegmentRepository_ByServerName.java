package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.MetricDBSegment;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

@View(name = "by_serverName", map = "function(doc) { if (doc.type=='MetricDBSegment') { emit(doc.serverName, doc._id) } }")
public class MetricDBSegmentRepository_ByServerName extends TypedCouchDbRepositorySupport<MetricDBSegment> {

    public MetricDBSegmentRepository_ByServerName(CouchDbConnector db) {
        super(MetricDBSegment.class, db, "MetricDBSegment_ByServerName");
        initStandardDesignDocument();
    }

    
    public List<MetricDBSegment> findLastestMetricDBs(String serverName) {
    	
        List<MetricDBSegment> dbSegmentList =  queryView("by_serverName", serverName);
        if (dbSegmentList.size() > 0)
        	return dbSegmentList;
        else
        	return null;
    }
    
    
}
