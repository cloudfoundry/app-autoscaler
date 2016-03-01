package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.collection;

import java.util.ArrayList;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.BoundApp;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.BoundAppRepository_All;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.BoundAppRepository_ByAppId;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.BoundAppRepository_ByServerName;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.BoundAppRepository_ByServiceId;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.BoundAppRepository_ByServiceId_AppId;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;

public class BoundAppRepositoryCollection extends TypedRepoCollection<BoundApp> {

	private static final Logger logger = Logger
			.getLogger(BoundAppRepositoryCollection.class);

	private BoundAppRepository_All boundAppRepo_all;
	private BoundAppRepository_ByAppId boundAppRepo_byAppId;
	private BoundAppRepository_ByServerName boundAppRepo_byServerName;
	private BoundAppRepository_ByServiceId_AppId boundAppRepo_byServiceId_AppId;
	private BoundAppRepository_ByServiceId boundAppRepo_byServiceId;

	public BoundAppRepositoryCollection(CouchDbConnector db) {
		boundAppRepo_all = new BoundAppRepository_All(db);
		boundAppRepo_byAppId = new BoundAppRepository_ByAppId(db);
		boundAppRepo_byServerName = new BoundAppRepository_ByServerName(db);
		boundAppRepo_byServiceId_AppId = new BoundAppRepository_ByServiceId_AppId(
				db);
		boundAppRepo_byServiceId = new BoundAppRepository_ByServiceId(db);

	}

	public BoundAppRepositoryCollection(CouchDbConnector db,
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

		repoList.add(boundAppRepo_all);
		repoList.add(boundAppRepo_byAppId);
		repoList.add(boundAppRepo_byServerName);
		repoList.add(boundAppRepo_byServiceId_AppId);
		repoList.add(boundAppRepo_byServiceId);
		return repoList;

	}

	@Override
	public TypedCouchDbRepositorySupport getDefaultRepo() {
		return boundAppRepo_all;
	}

	public List<BoundApp> findByServiceId(String serviceId) throws Exception {
		try {
			return boundAppRepo_byServiceId.findByServiceId(serviceId);
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

	public BoundApp findByAppId(String appId) throws Exception {
		try {
			return boundAppRepo_byAppId.findByAppId(appId);
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

	public List<BoundApp> findByServiceIdAndAppId(String serviceId, String appId)
			throws Exception {
		try {
			return boundAppRepo_byServiceId_AppId.findByServiceIdAndAppId(
					serviceId, appId);
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

	public List<BoundApp> getAllRecords() {
		try {
			return this.boundAppRepo_all.getAllRecords();
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

	
	public void removeByServiceIdAndAppId(String serviceId, String appId)
			throws Exception {
		List<BoundApp> apps = findByServiceIdAndAppId(serviceId, appId);
		for (BoundApp app : apps) {
			remove(app);
		}
	}

	public void updateByServiceIdAndAppId(String serviceId, String appId,
			String appType, String appName, String serverName, boolean insertIfNotFound)
			throws Exception {
		List<BoundApp> apps = findByServiceIdAndAppId(serviceId, appId);
		if (apps.size() > 0) {
			for (BoundApp app : apps) {
				if (appType != null
						&& appName != null
						&& (!appType.equals(app.getAppType()) || !appName
								.equals(app.getAppName()))) {
					app.setAppType(appType);
					app.setAppName(appName);
					app.setServerName(serverName);
					try {
						update(app);
					} catch (org.ektorp.UpdateConflictException e) {
						// ignore
					}
				}
			}
		} else if (insertIfNotFound) {
			BoundApp app = new BoundApp(appId, serviceId, appType, appName);
			app.setServerName(serverName);
			add(app);
		}
	}

	public List<BoundApp> getAllBoundApps(String serverName) {
		try {
			return boundAppRepo_byServerName.getAllBoundApps(serverName);
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

	public List<BoundApp> findDuplicatedByAppId(String appId) {
		try {
			return boundAppRepo_byAppId.findDuplicateByAppId(appId);
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

	public void removeMultipleByAppId(String appId) {
		if (appId == null)
			return;
		List<BoundApp> apps = findDuplicatedByAppId(appId);
		if (apps == null || apps.size() <= 1) {
			return;
		}
		for (int i = 1; i < apps.size(); i++)
			remove(apps.get(i));
	}

	public void removeByAppId(String appId) throws Exception {
		BoundApp app = findByAppId(appId);
		remove(app);
	}

}
