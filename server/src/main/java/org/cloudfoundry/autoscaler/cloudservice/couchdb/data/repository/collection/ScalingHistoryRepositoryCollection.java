package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.collection;

import java.util.ArrayList;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.ScalingHistory;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.ScalingHistoryRepository_All;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.ScalingHistoryRepository_ByScalingTime;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;

public class ScalingHistoryRepositoryCollection extends
		TypedRepoCollection<ScalingHistory> {

	private static final Logger logger = Logger
			.getLogger(ScalingHistoryRepositoryCollection.class);

	private ScalingHistoryRepository_All scalingHistoryRepo_all;
	private ScalingHistoryRepository_ByScalingTime scalingHistoryRepo_byScalingTime;

	public ScalingHistoryRepositoryCollection(CouchDbConnector db) {
		scalingHistoryRepo_all = new ScalingHistoryRepository_All(db);
		scalingHistoryRepo_byScalingTime = new ScalingHistoryRepository_ByScalingTime(
				db);

	}

	public ScalingHistoryRepositoryCollection(CouchDbConnector db,
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

		repoList.add(scalingHistoryRepo_all);
		repoList.add(scalingHistoryRepo_byScalingTime);
		return repoList;
	}

	@Override
	public TypedCouchDbRepositorySupport getDefaultRepo() {
		return scalingHistoryRepo_all;
	}

	public List<ScalingHistory> findByScalingTime(String appId, long startTime,
			long endTime) {
		try {
			return scalingHistoryRepo_byScalingTime.findByScalingTime(appId,
					startTime, endTime);
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

	public List<ScalingHistory> getAllRecords() {
		try {
			return this.scalingHistoryRepo_all.getAllRecords();
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

}
