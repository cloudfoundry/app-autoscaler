package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.couchdb;

import java.util.ArrayList;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.ScalingHistoryDAO;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.ScalingHistory;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.ComplexKey;
import org.ektorp.CouchDbConnector;
import org.ektorp.ViewQuery;
import org.ektorp.support.View;

public class ScalingHistoryDAOImpl extends CommonDAOImpl implements ScalingHistoryDAO {

	@View(name = "byAll", map = "function(doc) { if (doc.type == 'ScalingHistory' ) emit( [doc.appId, doc.status], doc._id )}")
	public class ScalingHistoryRepository_All extends TypedCouchDbRepositorySupport<ScalingHistory> {

		public ScalingHistoryRepository_All(CouchDbConnector db) {
			super(ScalingHistory.class, db, "ScalingHistory_byAll");
		}

		public List<ScalingHistory> getAllRecords() {
			return queryView("byAll");
		}

	}

	@View(name = "findByScalingTime", map = "function(doc) { if (doc.type == 'ScalingHistory' && doc.appId &&doc.startTime"
			+ ") emit( [doc.appId, doc.startTime], doc._id )}")
	public class ScalingHistoryRepository_ByScalingTime extends TypedCouchDbRepositorySupport<ScalingHistory> {

		public ScalingHistoryRepository_ByScalingTime(CouchDbConnector db) {
			super(ScalingHistory.class, db, "ScalingHistory_ByScalingTime");
		}

		public List<ScalingHistory> findByScalingTime(String appId, long startTime, long endTime) {
			ComplexKey startKey = ComplexKey.of(appId, endTime);
			ComplexKey endKey = ComplexKey.of(appId, startTime);
			ViewQuery q = createQuery("findByScalingTime").includeDocs(true).startKey(startKey).endKey(endKey)
					.descending(true);

			List<ScalingHistory> returnvalue = null;
			String[] input = beforeConnection("QUERY",
					new String[] { "findByScalingTime", appId, String.valueOf(startTime), String.valueOf(endTime) });
			try {
				returnvalue = db.queryView(q, ScalingHistory.class);
			} catch (Exception e) {
				e.printStackTrace();
			}
			afterConnection(input);

			return returnvalue;
		}

	}

	private static final Logger logger = Logger.getLogger(ScalingHistoryDAOImpl.class);
	private ScalingHistoryRepository_All scalingHistoryAllRepo = null;
	private ScalingHistoryRepository_ByScalingTime scalingHistoryByScalingTimeRepo = null;

	public ScalingHistoryDAOImpl(CouchDbConnector db) {
		this.scalingHistoryAllRepo = new ScalingHistoryRepository_All(db);
		this.scalingHistoryByScalingTimeRepo = new ScalingHistoryRepository_ByScalingTime(db);
	}

	public ScalingHistoryDAOImpl(CouchDbConnector db, boolean initDesignDocument) {
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
	public List<ScalingHistory> findAll() {
		// TODO Auto-generated method stub
		return this.scalingHistoryAllRepo.getAllRecords();
	}

	@Override
	public List<ScalingHistory> findByScalingTime(String appId, long startTime, long endTime) {
		// TODO Auto-generated method stub
		return this.scalingHistoryByScalingTimeRepo.findByScalingTime(appId, startTime, endTime);
	}

	@SuppressWarnings("unchecked")
	@Override
	public <T> TypedCouchDbRepositorySupport<T> getDefaultRepo() {
		// TODO Auto-generated method stub
		return (TypedCouchDbRepositorySupport<T>) this.scalingHistoryAllRepo;
	}

	@SuppressWarnings("unchecked")
	@Override
	public <T> List<TypedCouchDbRepositorySupport<T>> getAllRepos() {
		// TODO Auto-generated method stub
		List<TypedCouchDbRepositorySupport<T>> repoList = new ArrayList<TypedCouchDbRepositorySupport<T>>();
		repoList.add((TypedCouchDbRepositorySupport<T>) this.scalingHistoryAllRepo);
		repoList.add((TypedCouchDbRepositorySupport<T>) this.scalingHistoryByScalingTimeRepo);
		return repoList;
	}
}
