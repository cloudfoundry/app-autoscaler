package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.couchdb;

import java.util.ArrayList;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.ServiceInstanceDAO;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.ServiceInstance;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

public class ServiceInstanceDAOImpl extends CommonDAOImpl implements ServiceInstanceDAO {
	@View(name = "by_serverUrl", map = "function(doc) { if(doc.type=='ServiceInstance_inBroker' && doc.serverUrl) {emit(doc.serverUrl, doc._id)} }")
	public class ServiceInstanceRepository_ByServerURL extends TypedCouchDbRepositorySupport<ServiceInstance> {

		public ServiceInstanceRepository_ByServerURL(CouchDbConnector db) {
			super(ServiceInstance.class, db, "ServiceInstance_ByServerURL");
			initStandardDesignDocument();
		}

		public List<ServiceInstance> findByServerUrl(String serverUrl) {
			return queryView("by_serverUrl", serverUrl);
		}

	}

	@View(name = "by_serviceId", map = "function(doc) { if(doc.type=='ServiceInstance_inBroker' && doc.serviceId) {emit(doc.serviceId, doc._id)} }")
	public class ServiceInstanceRepository_ByServiceId extends TypedCouchDbRepositorySupport<ServiceInstance> {

		public ServiceInstanceRepository_ByServiceId(CouchDbConnector db) {
			super(ServiceInstance.class, db, "ServiceInstance_ByServiceId");
			initStandardDesignDocument();
		}

		public List<ServiceInstance> findByServiceId(String serviceId) {
			return queryView("by_serviceId", serviceId);
		}

		public List<ServiceInstance> findByAll() {
			return queryView("by_serviceId");
		}

		public boolean updateServerURL(String serviceId, String serverURL) {

			List<ServiceInstance> serviceInstances = this.findByServiceId(serviceId);
			if (serviceInstances != null) {
				for (ServiceInstance serviceInstance : serviceInstances) {
					serviceInstance.setServerUrl(serverURL);
					update(serviceInstance);
				}
				return true;
			} else
				return false;

		}

	}

	private static final Logger logger = Logger.getLogger(ServiceInstanceDAOImpl.class);
	private ServiceInstanceRepository_ByServerURL serviceInstanceByServerUrlRepo = null;
	private ServiceInstanceRepository_ByServiceId serviceInstanceByServiceIdRepo = null;

	public ServiceInstanceDAOImpl(CouchDbConnector db) {
		this.serviceInstanceByServerUrlRepo = new ServiceInstanceRepository_ByServerURL(db);
		this.serviceInstanceByServiceIdRepo = new ServiceInstanceRepository_ByServiceId(db);
	}

	public ServiceInstanceDAOImpl(CouchDbConnector db, boolean initDesignDocument) {
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
	public List<ServiceInstance> findByServerUrl(String serverUrl) {
		// TODO Auto-generated method stub
		return this.serviceInstanceByServerUrlRepo.findByServerUrl(serverUrl);
	}

	@Override
	public List<ServiceInstance> findByServiceId(String serviceId) {
		// TODO Auto-generated method stub
		return this.serviceInstanceByServiceIdRepo.findByServiceId(serviceId);
	}

	@SuppressWarnings("unchecked")
	@Override
	public <T> TypedCouchDbRepositorySupport<T> getDefaultRepo() {
		// TODO Auto-generated method stub
		return (TypedCouchDbRepositorySupport<T>) this.serviceInstanceByServiceIdRepo;
	}

	@SuppressWarnings("unchecked")
	@Override
	public <T> List<TypedCouchDbRepositorySupport<T>> getAllRepos() {
		// TODO Auto-generated method stub
		List<TypedCouchDbRepositorySupport<T>> repoList = new ArrayList<TypedCouchDbRepositorySupport<T>>();
		repoList.add((TypedCouchDbRepositorySupport<T>) this.serviceInstanceByServerUrlRepo);
		repoList.add((TypedCouchDbRepositorySupport<T>) this.serviceInstanceByServiceIdRepo);
		return repoList;
	}

}
