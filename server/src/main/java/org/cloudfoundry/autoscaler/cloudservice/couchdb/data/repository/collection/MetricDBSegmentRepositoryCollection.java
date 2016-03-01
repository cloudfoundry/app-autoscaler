package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.collection;

import java.util.ArrayList;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.MetricDBSegment;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.MetricDBSegmentRepository_All;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.MetricDBSegmentRepository_ByPostfix;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.MetricDBSegmentRepository_ByServerName;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.MetricDBSegmentRepository_ByServerName_SegmentSeq;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;

public class MetricDBSegmentRepositoryCollection extends
		TypedRepoCollection<MetricDBSegment> {

	private static final Logger logger = Logger
			.getLogger(MetricDBSegmentRepositoryCollection.class);

	private MetricDBSegmentRepository_All metricDBSegmentRepo_all;
	private MetricDBSegmentRepository_ByPostfix metricDBSegmentRepo_byPostfix;
	private MetricDBSegmentRepository_ByServerName metricDBSegmentRepo_byServerName;
	private MetricDBSegmentRepository_ByServerName_SegmentSeq metricDBSegmentRepo_byServerName_SegmentSeq;

	public MetricDBSegmentRepositoryCollection(CouchDbConnector db) {
		metricDBSegmentRepo_all = new MetricDBSegmentRepository_All(db);
		metricDBSegmentRepo_byPostfix = new MetricDBSegmentRepository_ByPostfix(
				db);
		metricDBSegmentRepo_byServerName = new MetricDBSegmentRepository_ByServerName(
				db);
		metricDBSegmentRepo_byServerName_SegmentSeq = new MetricDBSegmentRepository_ByServerName_SegmentSeq(
				db);
	}

	public MetricDBSegmentRepositoryCollection(CouchDbConnector db,
			boolean initDesignDocument) {
		this(db);
		if (initDesignDocument)
			try {
				initAllRepos();
			} catch (Exception e) {
				logger.error(e.getMessage(), e);
			}
	}

	@Override
	public List<TypedCouchDbRepositorySupport> getAllRepos() {
		List<TypedCouchDbRepositorySupport> repoList = new ArrayList<TypedCouchDbRepositorySupport>();
		repoList.add(metricDBSegmentRepo_all);
		repoList.add(metricDBSegmentRepo_byPostfix);
		repoList.add(metricDBSegmentRepo_byServerName);
		repoList.add(metricDBSegmentRepo_byServerName_SegmentSeq);
		return repoList;

	}

	@Override
	public TypedCouchDbRepositorySupport getDefaultRepo() {
		return metricDBSegmentRepo_all;
	}

	public MetricDBSegment findByMetricDBPostfix(String MetricDBPostfix) {
		try {
			return metricDBSegmentRepo_byPostfix
					.findByMetricDBPostfix(MetricDBPostfix);
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

	public MetricDBSegment findByServerNameSegmentSeq(String serverName, int seq) {
		try {
			return metricDBSegmentRepo_byServerName_SegmentSeq
					.findByServerNameSegmentSeq(serverName, seq);
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

	public List<MetricDBSegment> findLastestMetricDBs(String serverName) {
		try {
			return metricDBSegmentRepo_byServerName
					.findLastestMetricDBs(serverName);
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

	public List<MetricDBSegment> getAllRecords() {
		try {
			return this.metricDBSegmentRepo_all.getAllRecords();
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

	
	public boolean updateMetricDBSegment(MetricDBSegment segment) {

		MetricDBSegment curSegment = findByMetricDBPostfix(segment
				.getMetricDBPostfix());
		if (curSegment != null) {
			curSegment.setEndTimestamp(segment.getEndTimestamp());
			try {
				update(curSegment);
				return true;

			} catch (org.ektorp.UpdateConflictException e) {
				logger.error("Failed to update metric segment record: " + segment.getMetricDBPostfix() ,e);
				return false;
			}
		}
		return false;
	}

	public void removeAll() {
		List<MetricDBSegment> dbSegmentList = metricDBSegmentRepo_all.getAll();
		for (MetricDBSegment seg : dbSegmentList)
			remove(seg);
	}

}
