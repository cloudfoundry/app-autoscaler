package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.couchdb;

import java.util.ArrayList;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.BoundAppDAO;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.BoundApp;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.ComplexKey;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

public class BoundAppDAOImpl extends CommonDAOImpl implements BoundAppDAO {

	@View(name = "byAll", map = "function(doc) { if (doc.type == 'BoundApp' ) emit( [doc.appId, doc.serverName], doc._id )}")
	public class BoundAppRepository_All extends TypedCouchDbRepositorySupport<BoundApp> {
		public BoundAppRepository_All(CouchDbConnector db) {
			super(BoundApp.class, db, "BoundApp_byAll");
		}

		public List<BoundApp> getAllRecords() {
			return queryView("byAll");
		}

	}

	@View(name = "by_appId", map = "function(doc) { if (doc.type=='BoundApp' && doc.appId) { emit([doc.appId], doc._id) } }")
	public class BoundAppRepository_ByAppId extends TypedCouchDbRepositorySupport<BoundApp> {
		public BoundAppRepository_ByAppId(CouchDbConnector db) {
			super(BoundApp.class, db, "BoundApp_ByAppId");
		}

		public BoundApp findByAppId(String appId) throws Exception {
			List<BoundApp> apps = queryView("by_appId", ComplexKey.of(appId));
			if (apps == null || apps.size() == 0) {
				return null;
			}
			return apps.get(0);
		}

		public List<BoundApp> findDuplicateByAppId(String appId) {
			if (appId == null)
				return null;
			List<BoundApp> apps = queryView("by_appId", ComplexKey.of(appId));
			if (apps == null || apps.size() == 0) {
				return null;
			}
			return apps;
		}
	}

	@View(name = "by_servername", map = "function(doc) { if (doc.type=='BoundApp' && doc.serverName) { emit([doc.serverName], doc._id) } }")
	public class BoundAppRepository_ByServerName extends TypedCouchDbRepositorySupport<BoundApp> {
		public BoundAppRepository_ByServerName(CouchDbConnector db) {
			super(BoundApp.class, db, "BoundApp_ByServerName");
		}

		public List<BoundApp> getAllBoundApps(String serverName) {
			ComplexKey key = ComplexKey.of(serverName);
			return queryView("by_servername", key);
		}

	}

	@View(name = "by_serviceId_appId", map = "function(doc) { if (doc.type=='BoundApp' && doc.serviceId && doc.appId) { emit([doc.serviceId, doc.appId], doc._id) } }")
	public class BoundAppRepository_ByServiceId_AppId extends TypedCouchDbRepositorySupport<BoundApp> {
		public BoundAppRepository_ByServiceId_AppId(CouchDbConnector db) {
			super(BoundApp.class, db, "BoundApp_ByServiceId_AppId");
		}

		public List<BoundApp> findByServiceIdAndAppId(String serviceId, String appId) throws Exception {
			ComplexKey key = ComplexKey.of(serviceId, appId);
			return queryView("by_serviceId_appId", key);
		}

	}

	@View(name = "by_serviceId", map = "function(doc) { if (doc.type=='BoundApp' && doc.serviceId) { emit([doc.serviceId], doc._id) } }")
	public class BoundAppRepository_ByServiceId extends TypedCouchDbRepositorySupport<BoundApp> {
		public BoundAppRepository_ByServiceId(CouchDbConnector db) {
			super(BoundApp.class, db, "BoundApp_ByServiceId");
		}

		public List<BoundApp> findByServiceId(String serviceId) throws Exception {
			ComplexKey key = ComplexKey.of(serviceId);
			return queryView("by_serviceId", key);
		}

	}

	private static final Logger logger = Logger.getLogger(BoundAppDAOImpl.class);
	private BoundAppRepository_All boundAppRepoAll = null;
	private BoundAppRepository_ByAppId boundAppRepoByAppId = null;
	private BoundAppRepository_ByServerName boundAppRepoByServerName = null;
	private BoundAppRepository_ByServiceId_AppId boundAppRepoByServiceIdAppId = null;
	private BoundAppRepository_ByServiceId boundAppRepoByServiceId = null;

	public BoundAppDAOImpl(CouchDbConnector db) {

		this.boundAppRepoAll = new BoundAppRepository_All(db);
		this.boundAppRepoByAppId = new BoundAppRepository_ByAppId(db);
		this.boundAppRepoByServerName = new BoundAppRepository_ByServerName(db);
		this.boundAppRepoByServiceIdAppId = new BoundAppRepository_ByServiceId_AppId(db);
		this.boundAppRepoByServiceId = new BoundAppRepository_ByServiceId(db);

	}

	public BoundAppDAOImpl(CouchDbConnector db, boolean initDesignDocument) {
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
	public List<BoundApp> findAll() {
		// TODO Auto-generated method stub
		return this.boundAppRepoAll.getAllRecords();
	}

	@Override
	public BoundApp findByAppId(String appId) throws Exception {
		// TODO Auto-generated method stub
		return this.boundAppRepoByAppId.findByAppId(appId);
	}

	@Override
	public List<BoundApp> findByServerName(String serverName) {
		// TODO Auto-generated method stub
		return this.boundAppRepoByServerName.getAllBoundApps(serverName);
	}

	@Override
	public List<BoundApp> findByServiceIdAndAppId(String serviceId, String appId) throws Exception {
		// TODO Auto-generated method stub
		return this.boundAppRepoByServiceIdAppId.findByServiceIdAndAppId(serviceId, appId);
	}

	@Override
	public List<BoundApp> findByServiceId(String serviceId) throws Exception {
		// TODO Auto-generated method stub
		return this.boundAppRepoByServiceId.findByServiceId(serviceId);
	}

	@Override
	public List<BoundApp> getAllBoundApps(String serverName) {
		return this.boundAppRepoByServerName.getAllBoundApps(serverName);
	}

	@Override
	public void removeByServiceIdAndAppId(String serviceId, String appId) throws Exception {
		List<BoundApp> apps = findByServiceIdAndAppId(serviceId, appId);
		for (BoundApp app : apps) {
			remove(app);
		}
	}

	@Override
	public void updateByServiceIdAndAppId(String serviceId, String appId, String appType, String appName,
			String serverName, boolean insertIfNotFound) throws Exception {
		List<BoundApp> apps = findByServiceIdAndAppId(serviceId, appId);
		if (apps.size() > 0) {
			for (BoundApp app : apps) {
				if (appType != null && appName != null
						&& (!appType.equals(app.getAppType()) || !appName.equals(app.getAppName()))) {
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

	@SuppressWarnings("unchecked")
	@Override
	public <T> TypedCouchDbRepositorySupport<T> getDefaultRepo() {
		// TODO Auto-generated method stub
		return (TypedCouchDbRepositorySupport<T>) this.boundAppRepoAll;
	}

	@SuppressWarnings("unchecked")
	@Override
	public <T> List<TypedCouchDbRepositorySupport<T>> getAllRepos() {
		// TODO Auto-generated method stub
		List<TypedCouchDbRepositorySupport<T>> repoList = new ArrayList<TypedCouchDbRepositorySupport<T>>();
		repoList.add((TypedCouchDbRepositorySupport<T>) this.boundAppRepoAll);
		repoList.add((TypedCouchDbRepositorySupport<T>) this.boundAppRepoByAppId);
		repoList.add((TypedCouchDbRepositorySupport<T>) this.boundAppRepoByServerName);
		repoList.add((TypedCouchDbRepositorySupport<T>) this.boundAppRepoByServiceId);
		repoList.add((TypedCouchDbRepositorySupport<T>) this.boundAppRepoByServiceIdAppId);
		return repoList;
	}

}
