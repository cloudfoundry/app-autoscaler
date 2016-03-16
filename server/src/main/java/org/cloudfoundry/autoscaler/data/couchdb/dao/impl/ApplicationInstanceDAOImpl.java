package org.cloudfoundry.autoscaler.data.couchdb.dao.impl;

import java.util.ArrayList;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.data.couchdb.dao.ApplicationInstanceDAO;
import org.cloudfoundry.autoscaler.data.couchdb.dao.base.TypedCouchDbRepositorySupport;
import org.cloudfoundry.autoscaler.data.couchdb.document.ApplicationInstance;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

public class ApplicationInstanceDAOImpl extends CommonDAOImpl implements ApplicationInstanceDAO {
	@View(name = "by_appId", map = "function(doc) { if(doc.type=='ApplicationInstance_inBroker' && doc.appId) {emit(doc.appId, doc._id)} }")
	private static class ApplicationInstanceRepository_ByAppId
			extends TypedCouchDbRepositorySupport<ApplicationInstance> {

		public ApplicationInstanceRepository_ByAppId(CouchDbConnector db) {
			super(ApplicationInstance.class, db, "ApplicationInstance_ByAppId");
			initStandardDesignDocument();
		}

		public List<ApplicationInstance> findByAppId(String appId) {
			return queryView("by_appId", appId);
		}

		public List<ApplicationInstance> findByAll() {
			return queryView("by_appId");
		}
	}

	@View(name = "by_bindingId", map = "function(doc) { if(doc.type=='ApplicationInstance_inBroker' && doc.bindingId) {emit(doc.bindingId, doc._id)} }")
	private static class ApplicationInstanceRepository_ByBindingId
			extends TypedCouchDbRepositorySupport<ApplicationInstance> {

		public ApplicationInstanceRepository_ByBindingId(CouchDbConnector db) {
			super(ApplicationInstance.class, db, "ApplicationInstance_ByBindingId");
			initStandardDesignDocument();
		}

		public List<ApplicationInstance> findByBindingId(String bindingId) {
			return queryView("by_bindingId", bindingId);
		}

	}

	@View(name = "by_serviceId", map = "function(doc) { if(doc.type=='ApplicationInstance_inBroker' && doc.serviceId) {emit(doc.serviceId, doc._id)} }")
	private static class ApplicationInstanceRepository_ByServiceId
			extends TypedCouchDbRepositorySupport<ApplicationInstance> {

		public ApplicationInstanceRepository_ByServiceId(CouchDbConnector db) {
			super(ApplicationInstance.class, db, "ApplicationInstance_ByServiceId");
			initStandardDesignDocument();
		}

		public List<ApplicationInstance> findByServiceId(String serviceId) {
			return queryView("by_serviceId", serviceId);
		}

	}

	private static final Logger logger = Logger.getLogger(ApplicationInstanceDAOImpl.class);
	private ApplicationInstanceRepository_ByAppId instanceRepoByAppId = null;
	private ApplicationInstanceRepository_ByBindingId instanceRepoByBindingId = null;
	private ApplicationInstanceRepository_ByServiceId instanceRepoByServiceId = null;

	public ApplicationInstanceDAOImpl(CouchDbConnector db) {

		this.instanceRepoByAppId = new ApplicationInstanceRepository_ByAppId(db);
		this.instanceRepoByBindingId = new ApplicationInstanceRepository_ByBindingId(db);
		this.instanceRepoByServiceId = new ApplicationInstanceRepository_ByServiceId(db);

	}

	public ApplicationInstanceDAOImpl(CouchDbConnector db, boolean initDesignDocument) {
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
	public List<ApplicationInstance> findAll() {
		// TODO Auto-generated method stub
		return this.instanceRepoByAppId.findByAll();
	}

	@Override
	public List<ApplicationInstance> findByAppId(String appId) {
		// TODO Auto-generated method stub
		return this.instanceRepoByAppId.findByAppId(appId);
	}

	@Override
	public List<ApplicationInstance> findByBindId(String bindId) {
		// TODO Auto-generated method stub
		return this.instanceRepoByBindingId.findByBindingId(bindId);
	}

	@Override
	public List<ApplicationInstance> findByServiceId(String serviceId) {
		// TODO Auto-generated method stub
		return this.instanceRepoByServiceId.findByServiceId(serviceId);
	}

	@SuppressWarnings("unchecked")
	@Override
	public <T> TypedCouchDbRepositorySupport<T> getDefaultRepo() {
		// TODO Auto-generated method stub
		return (TypedCouchDbRepositorySupport<T>) this.instanceRepoByAppId;
	}

	@SuppressWarnings("unchecked")
	@Override
	public <T> List<TypedCouchDbRepositorySupport<T>> getAllRepos() {
		// TODO Auto-generated method stub
		List<TypedCouchDbRepositorySupport<T>> repoList = new ArrayList<TypedCouchDbRepositorySupport<T>>();
		repoList.add((TypedCouchDbRepositorySupport<T>) this.instanceRepoByAppId);
		repoList.add((TypedCouchDbRepositorySupport<T>) this.instanceRepoByBindingId);
		repoList.add((TypedCouchDbRepositorySupport<T>) this.instanceRepoByServiceId);
		return repoList;
	}

}
