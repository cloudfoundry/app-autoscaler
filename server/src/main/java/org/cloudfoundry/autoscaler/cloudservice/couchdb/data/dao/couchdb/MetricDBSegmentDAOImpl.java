package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.couchdb;

import java.util.ArrayList;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.MetricDBSegmentDAO;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.MetricDBSegment;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.ComplexKey;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

public class MetricDBSegmentDAOImpl extends CommonDAOImpl implements MetricDBSegmentDAO {
	@View(name = "byAll", map = "function(doc) { if (doc.type == 'MetricDBSegment' ) emit([doc.metricDBPostfix, doc.serverName, doc.segmentSeq], doc._id)}")
	private static class MetricDBSegmentRepository_All extends TypedCouchDbRepositorySupport<MetricDBSegment> {

		public MetricDBSegmentRepository_All(CouchDbConnector db) {
			super(MetricDBSegment.class, db, "MetricDBSegment_byAll");
		}

		public List<MetricDBSegment> getAllRecords() {
			return queryView("byAll");
		}
	}

	@View(name = "by_metricDBPostfix", map = "function(doc) { if (doc.type=='MetricDBSegment' && doc.metricDBPostfix) { emit(doc.metricDBPostfix, doc._id) } }")
	private static class MetricDBSegmentRepository_ByPostfix extends TypedCouchDbRepositorySupport<MetricDBSegment> {

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

	@View(name = "by_serverName_segmentSeq", map = "function(doc) { if (doc.type=='MetricDBSegment') { emit([doc.serverName, doc.segmentSeq], doc._id) } }")
	private static class MetricDBSegmentRepository_ByServerName_SegmentSeq
			extends TypedCouchDbRepositorySupport<MetricDBSegment> {

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

	@View(name = "by_serverName", map = "function(doc) { if (doc.type=='MetricDBSegment') { emit(doc.serverName, doc._id) } }")
	private static class MetricDBSegmentRepository_ByServerName extends TypedCouchDbRepositorySupport<MetricDBSegment> {

		public MetricDBSegmentRepository_ByServerName(CouchDbConnector db) {
			super(MetricDBSegment.class, db, "MetricDBSegment_ByServerName");
			initStandardDesignDocument();
		}

		public List<MetricDBSegment> findLastestMetricDBs(String serverName) {

			List<MetricDBSegment> dbSegmentList = queryView("by_serverName", serverName);
			if (dbSegmentList.size() > 0)
				return dbSegmentList;
			else
				return null;
		}

	}

	private static final Logger logger = Logger.getLogger(MetricDBSegmentDAOImpl.class);
	private MetricDBSegmentRepository_All metricDBSegmentAllRepo = null;
	private MetricDBSegmentRepository_ByPostfix metricDBSegmentByPostfixRepo = null;
	private MetricDBSegmentRepository_ByServerName_SegmentSeq metricDBSegmentByServerNameSegmentSeqRepo = null;
	private MetricDBSegmentRepository_ByServerName metricDBSegmentByServerName = null;

	public MetricDBSegmentDAOImpl(CouchDbConnector db) {
		this.metricDBSegmentAllRepo = new MetricDBSegmentRepository_All(db);
		this.metricDBSegmentByPostfixRepo = new MetricDBSegmentRepository_ByPostfix(db);
		this.metricDBSegmentByServerNameSegmentSeqRepo = new MetricDBSegmentRepository_ByServerName_SegmentSeq(db);
		this.metricDBSegmentByServerName = new MetricDBSegmentRepository_ByServerName(db);
	}

	public MetricDBSegmentDAOImpl(CouchDbConnector db, boolean initDesignDocument) {
		this(db);
		if (initDesignDocument) {
			try {
				initAllRepos();
			} catch (Exception e) {
				logger.error(e.getMessage(), e);
			}
		}

	}

	@Override
	public List<MetricDBSegment> findAll() {
		// TODO Auto-generated method stub
		return this.metricDBSegmentAllRepo.getAllRecords();
	}

	@Override
	public MetricDBSegment findByPostfix(String MetricDBPostfix) {
		// TODO Auto-generated method stub
		return this.metricDBSegmentByPostfixRepo.findByMetricDBPostfix(MetricDBPostfix);
	}

	@Override
	public MetricDBSegment findByServerNameSegmentSeq(String serverName, int seq) {
		// TODO Auto-generated method stub
		return this.metricDBSegmentByServerNameSegmentSeqRepo.findByServerNameSegmentSeq(serverName, seq);
	}

	@Override
	public List<MetricDBSegment> findByServerName(String serverName) {
		// TODO Auto-generated method stub
		return this.metricDBSegmentByServerName.findLastestMetricDBs(serverName);
	}

	@Override
	public List<MetricDBSegment> findLastestMetricDBs(String serverName) {
		return this.metricDBSegmentByServerName.findLastestMetricDBs(serverName);
	}

	@Override
	public boolean updateMetricDBSegment(MetricDBSegment segment) {

		MetricDBSegment curSegment = findByMetricDBPostfix(segment.getMetricDBPostfix());
		if (curSegment != null) {
			curSegment.setEndTimestamp(segment.getEndTimestamp());
			try {
				update(curSegment);
				return true;

			} catch (org.ektorp.UpdateConflictException e) {
				logger.error("Failed to update metric segment record: " + segment.getMetricDBPostfix(), e);
				return false;
			}
		}
		return false;
	}

	@Override
	public MetricDBSegment findByMetricDBPostfix(String MetricDBPostfix) {
		try {
			return this.metricDBSegmentByPostfixRepo.findByMetricDBPostfix(MetricDBPostfix);
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

	@SuppressWarnings("unchecked")
	@Override
	public <T> TypedCouchDbRepositorySupport<T> getDefaultRepo() {
		// TODO Auto-generated method stub
		return (TypedCouchDbRepositorySupport<T>) this.metricDBSegmentAllRepo;
	}

	@SuppressWarnings("unchecked")
	@Override
	public <T> List<TypedCouchDbRepositorySupport<T>> getAllRepos() {
		// TODO Auto-generated method stub
		List<TypedCouchDbRepositorySupport<T>> repoList = new ArrayList<TypedCouchDbRepositorySupport<T>>();
		repoList.add((TypedCouchDbRepositorySupport<T>) this.metricDBSegmentAllRepo);
		repoList.add((TypedCouchDbRepositorySupport<T>) this.metricDBSegmentByPostfixRepo);
		repoList.add((TypedCouchDbRepositorySupport<T>) this.metricDBSegmentByServerName);
		repoList.add((TypedCouchDbRepositorySupport<T>) this.metricDBSegmentByServerNameSegmentSeqRepo);
		return repoList;
	}

}
