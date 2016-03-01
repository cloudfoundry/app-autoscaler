package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.collection;

import java.util.ArrayList;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.AppInstanceMetrics;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.AppInstanceMetricsRepository_All;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.AppInstanceMetricsRepository_ByAppId;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.AppInstanceMetricsRepository_ByAppIdBetween;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.AppInstanceMetricsRepository_ByServiceId_Before;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;

public class AppInstanceMetricsRepositoryCollection extends
		TypedRepoCollection<AppInstanceMetrics> {

	private static final Logger logger = Logger
			.getLogger(AppInstanceMetricsRepositoryCollection.class);

	private AppInstanceMetricsRepository_All appInstanceMetricsRepo_all;
	private AppInstanceMetricsRepository_ByAppId appInstanceMetricsRepo_byAppId;
	private AppInstanceMetricsRepository_ByAppIdBetween appInstanceMetricsRepo_byAppIdBetween;
	private AppInstanceMetricsRepository_ByServiceId_Before appInstanceMetricsRepo_byServiceIdAfter;

	public AppInstanceMetricsRepositoryCollection(CouchDbConnector db) {
		appInstanceMetricsRepo_all = new AppInstanceMetricsRepository_All(db);
		appInstanceMetricsRepo_byAppId = new AppInstanceMetricsRepository_ByAppId(
				db);
		appInstanceMetricsRepo_byAppIdBetween = new AppInstanceMetricsRepository_ByAppIdBetween(
				db);
		appInstanceMetricsRepo_byServiceIdAfter = new AppInstanceMetricsRepository_ByServiceId_Before(
				db);
	}

	public AppInstanceMetricsRepositoryCollection(CouchDbConnector db,
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
		repoList.add(appInstanceMetricsRepo_all);
		repoList.add(appInstanceMetricsRepo_byAppId);
		repoList.add(appInstanceMetricsRepo_byAppIdBetween);
		repoList.add(appInstanceMetricsRepo_byServiceIdAfter);
		return repoList;
	}

	@Override
	public TypedCouchDbRepositorySupport getDefaultRepo() {
		return appInstanceMetricsRepo_all;
	}

	public List<AppInstanceMetrics> findByAppId(String appId) {
		try {
			return appInstanceMetricsRepo_byAppId.findByAppId(appId);
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

	public List<AppInstanceMetrics> findByServiceIdBefore(String serviceId,
			long olderThan) throws Exception {
		try {
			return appInstanceMetricsRepo_byServiceIdAfter
					.findByServiceIdBefore(serviceId, olderThan);
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

	public List<AppInstanceMetrics> findByAppIdBetween(String appId,
			long startTimestamp, long endTimestamp) throws Exception {
		try {
			return appInstanceMetricsRepo_byAppIdBetween.findByAppIdBetween(
					appId, startTimestamp, endTimestamp);
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

	public List<AppInstanceMetrics> findByAppIdAfter(String appId,
			long timestamp) throws Exception {
		try {
			return findByAppIdBetween(appId, timestamp,
					System.currentTimeMillis());
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

	public List<AppInstanceMetrics> getAllRecords() {
		try {
			return this.appInstanceMetricsRepo_all.getAllRecords();
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}
	
}
