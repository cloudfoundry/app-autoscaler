package org.cloudfoundry.autoscaler.data.couchdb.dao.impl;

import java.util.ArrayList;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.data.couchdb.dao.ApplicationDAO;
import org.cloudfoundry.autoscaler.data.couchdb.dao.base.TypedCouchDbRepositorySupport;
import org.cloudfoundry.autoscaler.data.couchdb.document.Application;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

public class ApplicationDAOImpl extends CommonDAOImpl implements ApplicationDAO {

	@View(name = "byAll", map = "function(doc) { if (doc.type == 'Application' ) emit([doc.appId, doc.serviceId, doc.bindingId, doc.policyId, doc.state], doc._id )}")
	public class ApplicationRepository_All extends TypedCouchDbRepositorySupport<Application> {
		public ApplicationRepository_All(CouchDbConnector db) {
			super(Application.class, db, "Application_byAll");
		}

		public List<Application> getAllRecords() {
			return queryView("byAll");
		}

	}

	@View(name = "by_appId", map = "function(doc) { if(doc.type=='Application' && doc.appId) {emit(doc.appId, doc._id)} }")
	public class ApplicationRepository_ByAppId extends TypedCouchDbRepositorySupport<Application> {
		public ApplicationRepository_ByAppId(CouchDbConnector db) {
			super(Application.class, db, "Application_ByAppId");
		}

		public Application findByAppId(String appId) {
			if (appId == null)
				return null;
			List<Application> apps = queryView("by_appId", appId);
			if (apps == null || apps.size() == 0) {
				return null;
			}
			return apps.get(0);
		}

		public List<Application> findDupliateByAppId(String appId) {
			if (appId == null)
				return null;
			List<Application> apps = queryView("by_appId", appId);
			if (apps == null || apps.size() == 0) {
				return null;
			}
			return apps;
		}

	}

	@View(name = "by_bindingId", map = "function(doc) { if(doc.type=='Application' && doc.bindingId) {emit(doc.bindingId, doc._id)} }")
	public class ApplicationRepository_ByBindingId extends TypedCouchDbRepositorySupport<Application> {
		public ApplicationRepository_ByBindingId(CouchDbConnector db) {
			super(Application.class, db, "Application_ByBindingId");
		}

		public Application findByBindingId(String bindingId) {
			if (bindingId == null)
				return null;
			List<Application> apps = queryView("by_bindingId", bindingId);
			if (apps == null || apps.size() == 0) {
				return null;
			}
			return apps.get(0);

		}

	}

	@View(name = "by_policyId", map = "function(doc) { if(doc.type=='Application' && doc.policyId) {emit(doc.policyId, doc._id)} }")
	public class ApplicationRepository_ByPolicyId extends TypedCouchDbRepositorySupport<Application> {
		public ApplicationRepository_ByPolicyId(CouchDbConnector db) {
			super(Application.class, db, "Application_ByPolicyId");
		}

		public List<Application> findByPolicyId(String policyId) {
			return queryView("by_policyId", policyId);
		}

	}

	@View(name = "by_serviceId_state", map = "function(doc) { if (doc.type == 'Application' &&  doc.state != 'unbond' && doc.serviceId ) emit( doc.serviceId, doc._id )}")
	public class ApplicationRepository_ByServiceId_State extends TypedCouchDbRepositorySupport<Application> {
		public ApplicationRepository_ByServiceId_State(CouchDbConnector db) {
			super(Application.class, db, "Application_ByServiceId_State");
		}

		public List<Application> findByServiceIdAndState(String serviceId) {
			return queryView("by_serviceId_state", serviceId);
		}

	}

	@View(name = "by_serviceId", map = "function(doc) { if(doc.type=='Application' && doc.serviceId) {emit(doc.serviceId, doc._id)} }")
	public class ApplicationRepository_ByServiceId extends TypedCouchDbRepositorySupport<Application> {
		public ApplicationRepository_ByServiceId(CouchDbConnector db) {
			super(Application.class, db, "Application_ByServiceId");
		}

		public List<Application> findByServiceId(String serviceId) {
			return queryView("by_serviceId", serviceId);
		}

	}

	private static final Logger logger = Logger.getLogger(ApplicationDAOImpl.class);
	private ApplicationRepository_All appRepoAll = null;
	private ApplicationRepository_ByAppId appRepoByAppId = null;
	private ApplicationRepository_ByBindingId appRepoByBindingId = null;
	private ApplicationRepository_ByPolicyId appRepoByPolicyId = null;
	private ApplicationRepository_ByServiceId_State appRepoByServiceIdState = null;
	private ApplicationRepository_ByServiceId appRepoByServiceId = null;

	public ApplicationDAOImpl(CouchDbConnector db) {
		this.appRepoAll = new ApplicationRepository_All(db);
		this.appRepoByAppId = new ApplicationRepository_ByAppId(db);
		this.appRepoByBindingId = new ApplicationRepository_ByBindingId(db);
		this.appRepoByPolicyId = new ApplicationRepository_ByPolicyId(db);
		this.appRepoByServiceIdState = new ApplicationRepository_ByServiceId_State(db);
		this.appRepoByServiceId = new ApplicationRepository_ByServiceId(db);
	}

	public ApplicationDAOImpl(CouchDbConnector db, boolean initDesignDocument) {
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
	public List<Application> findAll() {
		// TODO Auto-generated method stub
		return this.appRepoAll.getAllRecords();
	}

	@Override
	public Application findByAppId(String appId) {
		// TODO Auto-generated method stub
		return this.appRepoByAppId.findByAppId(appId);
	}

	@Override
	public Application findByBindId(String bindId) {
		// TODO Auto-generated method stub
		return this.appRepoByBindingId.findByBindingId(bindId);
	}

	@Override
	public List<Application> findByPolicyId(String policyId) {
		// TODO Auto-generated method stub
		return this.appRepoByPolicyId.findByPolicyId(policyId);
	}

	@Override
	public List<Application> findByServiceIdState(String serviceId) {
		// TODO Auto-generated method stub
		return this.appRepoByServiceIdState.findByServiceIdAndState(serviceId);
	}

	@Override
	public List<Application> findByServiceId(String serviceId) {
		// TODO Auto-generated method stub
		return this.appRepoByServiceId.findByServiceId(serviceId);
	}

	@Override
	public List<Application> findByServiceIdAndState(String serviceId) {
		return this.appRepoByServiceIdState.findByServiceIdAndState(serviceId);
	}

	@SuppressWarnings("unchecked")
	@Override
	public <T> TypedCouchDbRepositorySupport<T> getDefaultRepo() {
		// TODO Auto-generated method stub
		return (TypedCouchDbRepositorySupport<T>) this.appRepoAll;
	}

	@SuppressWarnings("unchecked")
	@Override
	public <T> List<TypedCouchDbRepositorySupport<T>> getAllRepos() {
		// TODO Auto-generated method stub
		List<TypedCouchDbRepositorySupport<T>> repoList = new ArrayList<TypedCouchDbRepositorySupport<T>>();
		repoList.add((TypedCouchDbRepositorySupport<T>) this.appRepoAll);
		repoList.add((TypedCouchDbRepositorySupport<T>) this.appRepoByAppId);
		repoList.add((TypedCouchDbRepositorySupport<T>) this.appRepoByBindingId);
		repoList.add((TypedCouchDbRepositorySupport<T>) this.appRepoByPolicyId);
		repoList.add((TypedCouchDbRepositorySupport<T>) this.appRepoByServiceId);
		repoList.add((TypedCouchDbRepositorySupport<T>) this.appRepoByServiceIdState);
		return repoList;
	}
}
