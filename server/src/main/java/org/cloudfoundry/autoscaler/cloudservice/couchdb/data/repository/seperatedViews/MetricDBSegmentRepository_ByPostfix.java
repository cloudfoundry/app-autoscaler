package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.MetricDBSegment;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

@View(name = "by_metricDBPostfix", map = "function(doc) { if (doc.type=='MetricDBSegment' && doc.metricDBPostfix) { emit(doc.metricDBPostfix, doc._id) } }")
public class MetricDBSegmentRepository_ByPostfix extends TypedCouchDbRepositorySupport<MetricDBSegment> {

    public MetricDBSegmentRepository_ByPostfix(CouchDbConnector db) {
        super(MetricDBSegment.class, db, "MetricDBSegment_ByPostfix");
    }
    
    public MetricDBSegment findByMetricDBPostfix(String MetricDBPostfix) {
        List<MetricDBSegment> dbSegmentList = queryView("by_metricDBPostfix", MetricDBPostfix);
        if (dbSegmentList.size() > 0)
        	return dbSegmentList.get(0);
        else
        	return null;
    }
    
    
}
