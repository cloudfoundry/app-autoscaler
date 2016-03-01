package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.MetricDBSegment;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.ComplexKey;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

@View(name = "by_serverName_segmentSeq", map = "function(doc) { if (doc.type=='MetricDBSegment') { emit([doc.serverName, doc.segmentSeq], doc._id) } }")
public class MetricDBSegmentRepository_ByServerName_SegmentSeq extends TypedCouchDbRepositorySupport<MetricDBSegment> {

    public MetricDBSegmentRepository_ByServerName_SegmentSeq(CouchDbConnector db) {
        super(MetricDBSegment.class, db, "MetricDBSegment_ByServerName_SegmentSeq");
        initStandardDesignDocument();
    }
	
    public MetricDBSegment findByServerNameSegmentSeq(String serverName, int seq) {
        ComplexKey key = ComplexKey.of(serverName, seq);
        List<MetricDBSegment> dbSegmentList = queryView("by_serverName_segmentSeq", key);
        if (dbSegmentList.size() > 0)
        	return dbSegmentList.get(0);
        else
        	return null;
    }

    
    
}
