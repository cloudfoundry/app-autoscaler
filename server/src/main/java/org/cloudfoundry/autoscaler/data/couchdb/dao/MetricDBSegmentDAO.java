package org.cloudfoundry.autoscaler.data.couchdb.dao;

import java.util.List;

import org.cloudfoundry.autoscaler.data.couchdb.document.MetricDBSegment;

public interface MetricDBSegmentDAO extends CommonDAO {

	public List<MetricDBSegment> findAll();

	public MetricDBSegment findByPostfix(String MetricDBPostfix);

	public MetricDBSegment findByServerNameSegmentSeq(String serverName, int seq);

	public List<MetricDBSegment> findByServerName(String serverName);

	public List<MetricDBSegment> findLastestMetricDBs(String serverName);

	public boolean updateMetricDBSegment(MetricDBSegment segment);

	public MetricDBSegment findByMetricDBPostfix(String MetricDBPostfix);

}
