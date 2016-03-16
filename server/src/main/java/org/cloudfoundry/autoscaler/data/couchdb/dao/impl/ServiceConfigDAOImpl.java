package org.cloudfoundry.autoscaler.data.couchdb.dao.impl;

import java.util.ArrayList;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.data.couchdb.dao.ServiceConfigDAO;
import org.cloudfoundry.autoscaler.data.couchdb.dao.base.TypedCouchDbRepositorySupport;
import org.cloudfoundry.autoscaler.data.couchdb.document.ServiceConfig;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

public class ServiceConfigDAOImpl extends CommonDAOImpl implements ServiceConfigDAO {
	@View(name = "byAll", map = "function(doc) { if (doc.type == 'ServiceConfig' ) emit( [doc.serviceId], doc._id )}")
	public class ConfigRepository_All extends TypedCouchDbRepositorySupport<ServiceConfig> {

		public ConfigRepository_All(CouchDbConnector db) {
			super(ServiceConfig.class, db, "ServiceConfig_byAll");
		}

		public List<ServiceConfig> getAllRecords() {
			return queryView("byAll");
		}

	}

	private static final Logger logger = Logger.getLogger(ServiceConfigDAOImpl.class);
	private ConfigRepository_All configAllRepo = null;

	public ServiceConfigDAOImpl(CouchDbConnector db) {
		this.configAllRepo = new ConfigRepository_All(db);
	}

	public ServiceConfigDAOImpl(CouchDbConnector db, boolean initDesignDocument) {
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
	public List<ServiceConfig> findAll() {
		// TODO Auto-generated method stub
		return this.configAllRepo.getAllRecords();
	}

	@SuppressWarnings("unchecked")
	@Override
	public <T> TypedCouchDbRepositorySupport<T> getDefaultRepo() {
		// TODO Auto-generated method stub
		return (TypedCouchDbRepositorySupport<T>) this.configAllRepo;
	}

	@SuppressWarnings("unchecked")
	@Override
	public <T> List<TypedCouchDbRepositorySupport<T>> getAllRepos() {
		// TODO Auto-generated method stub
		List<TypedCouchDbRepositorySupport<T>> repoList = new ArrayList<TypedCouchDbRepositorySupport<T>>();
		repoList.add((TypedCouchDbRepositorySupport<T>) this.configAllRepo);
		return repoList;
	}

}
