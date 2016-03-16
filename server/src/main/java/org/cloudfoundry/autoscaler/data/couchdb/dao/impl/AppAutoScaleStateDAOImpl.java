package org.cloudfoundry.autoscaler.data.couchdb.dao.impl;

import java.util.ArrayList;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.data.couchdb.dao.AppAutoScaleStateDAO;
import org.cloudfoundry.autoscaler.data.couchdb.dao.base.TypedCouchDbRepositorySupport;
import org.cloudfoundry.autoscaler.data.couchdb.document.AppAutoScaleState;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

public class AppAutoScaleStateDAOImpl extends CommonDAOImpl implements AppAutoScaleStateDAO {

	@View(name = "by_appId", map = "function(doc) { if(doc.type=='AppAutoScaleState' && doc.appId) {emit(doc.appId, doc._id)} }")
	private static class ScalingStateRepository_ByAppId extends TypedCouchDbRepositorySupport<AppAutoScaleState> {

		public ScalingStateRepository_ByAppId(CouchDbConnector db) {
			super(AppAutoScaleState.class, db, "AppAutoScaleState_ByAppId");
		}

		public AppAutoScaleState findByAppId(String appId) {
			if (appId == null)
				return null;

			List<AppAutoScaleState> states = queryView("by_appId", appId);
			if (states == null || states.size() == 0) {
				return null;
			}
			return states.get(0);
		}

	}

	@View(name = "byAll", map = "function(doc) { if (doc.type == 'AppAutoScaleState' ) emit( [doc.appId, doc.instanceCountState], doc._id )}")
	private static class ScalingStateRepository_All extends TypedCouchDbRepositorySupport<AppAutoScaleState> {

		public ScalingStateRepository_All(CouchDbConnector db) {
			super(AppAutoScaleState.class, db, "AppAutoScaleState_byAll");
		}

		public List<AppAutoScaleState> getAllRecords() {
			return queryView("byAll");
		}

	}

	private static final Logger logger = Logger.getLogger(AppAutoScaleStateDAOImpl.class);
	private ScalingStateRepository_All stateRepoAll = null;
	private ScalingStateRepository_ByAppId stateRepoByAppId = null;

	public AppAutoScaleStateDAOImpl(CouchDbConnector db) {
		stateRepoAll = new ScalingStateRepository_All(db);
		stateRepoByAppId = new ScalingStateRepository_ByAppId(db);
	}

	public AppAutoScaleStateDAOImpl(CouchDbConnector db, boolean initDesignDocument) {
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
	public List<AppAutoScaleState> findAll() {
		// TODO Auto-generated method stub
		return this.stateRepoAll.getAllRecords();
	}

	@Override
	public AppAutoScaleState findByAppId(String appId) {
		// TODO Auto-generated method stub
		return this.stateRepoByAppId.findByAppId(appId);
	}

	@SuppressWarnings("unchecked")
	@Override
	public <T> TypedCouchDbRepositorySupport<T> getDefaultRepo() {
		// TODO Auto-generated method stub
		return (TypedCouchDbRepositorySupport<T>) this.stateRepoByAppId;
	}

	@SuppressWarnings("unchecked")
	@Override
	public <T> List<TypedCouchDbRepositorySupport<T>> getAllRepos() {
		// TODO Auto-generated method stub
		List<TypedCouchDbRepositorySupport<T>> repoList = new ArrayList<TypedCouchDbRepositorySupport<T>>();
		repoList.add((TypedCouchDbRepositorySupport<T>) this.stateRepoAll);
		repoList.add((TypedCouchDbRepositorySupport<T>) this.stateRepoByAppId);
		return repoList;
	}

}
