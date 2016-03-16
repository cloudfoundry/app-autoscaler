package org.cloudfoundry.autoscaler.data.couchdb.dao.impl;

import java.util.ArrayList;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.data.couchdb.dao.AppInstanceMetricsDAO;
import org.cloudfoundry.autoscaler.data.couchdb.dao.base.TypedCouchDbRepositorySupport;
import org.cloudfoundry.autoscaler.data.couchdb.document.AppInstanceMetrics;
import org.ektorp.ComplexKey;
import org.ektorp.CouchDbConnector;
import org.ektorp.ViewQuery;
import org.ektorp.support.View;

public class AppInstanceMetricsDAOImpl extends CommonDAOImpl implements AppInstanceMetricsDAO {
	@View(name = "byAll", map = "function(doc) { if (doc.type == 'AppInstanceMetrics' ) emit([doc.appId, doc.appType, doc.timestamp], doc._id)}")
	private static class AppInstanceMetricsRepository_All extends TypedCouchDbRepositorySupport<AppInstanceMetrics> {

		public AppInstanceMetricsRepository_All(CouchDbConnector db) {
			super(AppInstanceMetrics.class, db, "AppInstanceMetrics_byAll");
		}

		public List<AppInstanceMetrics> getAllRecords() {
			return queryView("byAll");
		}

	}

	@View(name = "by_appId", map = "function(doc) { if (doc.type=='AppInstanceMetrics' && doc.appId) { emit([doc.appId], doc._id) } }")
	private static class AppInstanceMetricsRepository_ByAppId
			extends TypedCouchDbRepositorySupport<AppInstanceMetrics> {

		public AppInstanceMetricsRepository_ByAppId(CouchDbConnector db) {
			super(AppInstanceMetrics.class, db, "AppInstanceMetrics_ByAppId");
		}

		public List<AppInstanceMetrics> findByAppId(String appId) {
			ComplexKey key = ComplexKey.of(appId);
			return queryView("by_appId", key);
		}

	}

	@View(name = "by_appId_between", map = "function(doc) { if (doc.type=='AppInstanceMetrics' && doc.appId && doc.timestamp) { emit([doc.appId, doc.timestamp], doc._id) } }")
	private static class AppInstanceMetricsRepository_ByAppIdBetween
			extends TypedCouchDbRepositorySupport<AppInstanceMetrics> {

		public AppInstanceMetricsRepository_ByAppIdBetween(CouchDbConnector db) {
			super(AppInstanceMetrics.class, db, "AppInstanceMetrics_ByAppIdBetween");
		}

		public List<AppInstanceMetrics> findByAppIdBetween(String appId, long startTimestamp, long endTimestamp)
				throws Exception {

			ComplexKey startKey = ComplexKey.of(appId, startTimestamp);
			ComplexKey endKey = ComplexKey.of(appId, endTimestamp);
			ViewQuery q = createQuery("by_appId_between").includeDocs(true).startKey(startKey).endKey(endKey);

			List<AppInstanceMetrics> returnvalue = null;
			String[] input = beforeConnection("QUERY", new String[] { "by_appId_between", appId,
					String.valueOf(startTimestamp), String.valueOf(endTimestamp) });
			try {
				returnvalue = db.queryView(q, AppInstanceMetrics.class);
			} catch (Exception e) {
				e.printStackTrace();
			}
			afterConnection(input);

			return returnvalue;
		}

	}

	@View(name = "by_serviceId_before", map = "function(doc) { if (doc.type=='AppInstanceMetrics' && doc.serviceId && doc.timestamp) { emit([ doc.serviceId, doc.timestamp], doc._id) } }")
	private static class AppInstanceMetricsRepository_ByServiceId_Before
			extends TypedCouchDbRepositorySupport<AppInstanceMetrics> {

		public AppInstanceMetricsRepository_ByServiceId_Before(CouchDbConnector db) {
			super(AppInstanceMetrics.class, db, "AppInstanceMetrics_ByServiceId");
		}

		public List<AppInstanceMetrics> findByServiceIdBefore(String serviceId, long olderThan) throws Exception {
			ComplexKey startKey = ComplexKey.of(serviceId, 0);
			ComplexKey endKey = ComplexKey.of(serviceId, olderThan);
			ViewQuery q = createQuery("by_serviceId_before").includeDocs(true).startKey(startKey).endKey(endKey);

			List<AppInstanceMetrics> returnvalue = null;
			String[] input = beforeConnection("QUERY",
					new String[] { "by_serviceId_before", serviceId, String.valueOf(0), String.valueOf(olderThan) });
			try {
				returnvalue = db.queryView(q, AppInstanceMetrics.class);
			} catch (Exception e) {
				e.printStackTrace();
			}
			afterConnection(input);

			return returnvalue;
		}

	}

	private static final Logger logger = Logger.getLogger(AppInstanceMetricsDAOImpl.class);
	private AppInstanceMetricsRepository_All metricsRepoAll;
	private AppInstanceMetricsRepository_ByAppId metricsRepoByAppId;
	private AppInstanceMetricsRepository_ByAppIdBetween metricsRepoByAppIdBetween;
	private AppInstanceMetricsRepository_ByServiceId_Before metricsRepoByServiceIdBefore;

	public AppInstanceMetricsDAOImpl(CouchDbConnector db) {
		metricsRepoAll = new AppInstanceMetricsRepository_All(db);
		metricsRepoByAppId = new AppInstanceMetricsRepository_ByAppId(db);
		metricsRepoByAppIdBetween = new AppInstanceMetricsRepository_ByAppIdBetween(db);
		metricsRepoByServiceIdBefore = new AppInstanceMetricsRepository_ByServiceId_Before(db);
	}

	public AppInstanceMetricsDAOImpl(CouchDbConnector db, boolean initDesignDocument) {
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
	public List<AppInstanceMetrics> findAll() {
		// TODO Auto-generated method stub
		return this.metricsRepoAll.getAllRecords();
	}

	@Override
	public List<AppInstanceMetrics> findByAppId(String appId) {
		// TODO Auto-generated method stub
		return this.metricsRepoByAppId.findByAppId(appId);
	}

	@Override
	public List<AppInstanceMetrics> findByAppIdBetween(String appId, long startTimestamp, long endTimestamp)
			throws Exception {
		// TODO Auto-generated method stub
		return this.metricsRepoByAppIdBetween.findByAppIdBetween(appId, startTimestamp, endTimestamp);
	}

	@Override
	public List<AppInstanceMetrics> findByServiceIdBefore(String serviceId, long olderThan) throws Exception {
		// TODO Auto-generated method stub
		return this.metricsRepoByServiceIdBefore.findByServiceIdBefore(serviceId, olderThan);
	}

	@Override
	public List<AppInstanceMetrics> findByAppIdAfter(String appId, long timestamp) throws Exception {
		try {
			return findByAppIdBetween(appId, timestamp, System.currentTimeMillis());
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

	@SuppressWarnings("unchecked")
	@Override
	public <T> TypedCouchDbRepositorySupport<T> getDefaultRepo() {
		// TODO Auto-generated method stub
		return (TypedCouchDbRepositorySupport<T>) this.metricsRepoAll;
	}

	@SuppressWarnings("unchecked")
	@Override
	public <T> List<TypedCouchDbRepositorySupport<T>> getAllRepos() {
		// TODO Auto-generated method stub
		List<TypedCouchDbRepositorySupport<T>> repoList = new ArrayList<TypedCouchDbRepositorySupport<T>>();
		repoList.add((TypedCouchDbRepositorySupport<T>) this.metricsRepoAll);
		repoList.add((TypedCouchDbRepositorySupport<T>) this.metricsRepoByAppId);
		repoList.add((TypedCouchDbRepositorySupport<T>) this.metricsRepoByAppIdBetween);
		repoList.add((TypedCouchDbRepositorySupport<T>) this.metricsRepoByServiceIdBefore);
		return repoList;
	}

}
