package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.MetricDBSegment;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

@View(name = "byAll", map = "function(doc) { if (doc.type == 'MetricDBSegment' ) emit([doc.metricDBPostfix, doc.serverName, doc.segmentSeq], doc._id)}")
public class MetricDBSegmentRepository_All extends TypedCouchDbRepositorySupport<MetricDBSegment> {

    public MetricDBSegmentRepository_All(CouchDbConnector db) {
        super(MetricDBSegment.class, db, "MetricDBSegment_byAll");
    }
    
    public List<MetricDBSegment> getAllRecords(){
    	return queryView("byAll");
    }
}
