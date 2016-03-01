package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.collection;

import java.util.ArrayList;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.AppAutoScaleState;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.ScalingStateRepository_All;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.ScalingStateRepository_ByAppId;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;

public class ScalingStateRepositoryCollection extends
		TypedRepoCollection<AppAutoScaleState> {

	private static final Logger logger = Logger
			.getLogger(ScalingStateRepositoryCollection.class);

	private ScalingStateRepository_All scalingStateRepo_all;
	private ScalingStateRepository_ByAppId scalingStateRepo_byAppId;

	public ScalingStateRepositoryCollection(CouchDbConnector db) {
		scalingStateRepo_all = new ScalingStateRepository_All(db);
		scalingStateRepo_byAppId = new ScalingStateRepository_ByAppId(db);

	}

	public ScalingStateRepositoryCollection(CouchDbConnector db,
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

		repoList.add(scalingStateRepo_all);
		repoList.add(scalingStateRepo_byAppId);
		return repoList;
	}

	@Override
	public TypedCouchDbRepositorySupport getDefaultRepo() {
		return scalingStateRepo_all;
	}

	public AppAutoScaleState findByAppId(String appId) {
		try {
			return scalingStateRepo_byAppId.findByAppId(appId);
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

	public List<AppAutoScaleState> getAllRecords() {
		try {
			return this.scalingStateRepo_all.getAllRecords();
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

}
