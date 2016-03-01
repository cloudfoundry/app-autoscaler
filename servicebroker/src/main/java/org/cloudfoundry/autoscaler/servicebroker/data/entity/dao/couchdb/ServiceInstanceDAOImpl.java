package org.cloudfoundry.autoscaler.servicebroker.data.entity.dao.couchdb;

import java.util.List;

import org.cloudfoundry.autoscaler.servicebroker.data.entity.ServiceInstance;
import org.cloudfoundry.autoscaler.servicebroker.data.entity.dao.ServiceInstanceDAO;
import org.ektorp.CouchDbConnector;
import org.ektorp.ViewQuery;
import org.ektorp.support.GenerateView;
import org.ektorp.support.View;


public class ServiceInstanceDAOImpl extends CommonDAOImpl implements ServiceInstanceDAO{

	@View ( name = "by_serverUrl", map = "function(doc) { if(doc.type=='ServiceInstance_inBroker' && doc.serverUrl) {emit(doc.serverUrl, doc._id)} }")
	private static class ServiceInstanceRepository_ByServerURL extends TypedCouchdbRepositorySupport<ServiceInstance> {

		public ServiceInstanceRepository_ByServerURL(CouchDbConnector db) {
			super(ServiceInstance.class, db, "ServiceInstance_ByServerURL"); 
			initStandardDesignDocument();
		}

		private List<ServiceInstance> findByServerUrl(String serverUrl) {
			return queryView("by_serverUrl", serverUrl);
		}
		
	    private int sizeOfServerUrl(String serverUrl) {
	        ViewQuery q = createQuery("by_serverUrl").includeDocs(false).key(serverUrl);
	        return db.queryView(q, String.class).size();
	    }
	}

	@View ( name = "by_serviceId", map = "function(doc) { if(doc.type=='ServiceInstance_inBroker' && doc.serviceId) {emit(doc.serviceId, doc._id)} }")
	private static class ServiceInstanceRepository_ByServiceId extends TypedCouchdbRepositorySupport<ServiceInstance> {

		public ServiceInstanceRepository_ByServiceId(CouchDbConnector db) {
			super(ServiceInstance.class, db, "ServiceInstance_ByServiceId");  
			initStandardDesignDocument();
		}

		private List<ServiceInstance> findByServiceId(String serviceId) {
			return queryView("by_serviceId", serviceId); 
		}
	}

    private	ServiceInstanceRepository_ByServerURL serviceRepo_byServerURL = null;                                            
    private	ServiceInstanceRepository_ByServiceId serviceRepo_byServiceId = null;                                       

	public ServiceInstanceDAOImpl(CouchDbConnector db) {
		serviceRepo_byServerURL = new ServiceInstanceRepository_ByServerURL(db);
		serviceRepo_byServiceId = new ServiceInstanceRepository_ByServiceId(db);
	}

	@Override
	public <T> TypedCouchdbRepositorySupport<T> getDefaultRepo() {
		return (TypedCouchdbRepositorySupport<T>) this.serviceRepo_byServiceId;
	}

	@Override
	public List<ServiceInstance> findByServerURL(String serverURL) {
		return this.serviceRepo_byServerURL.findByServerUrl(serverURL);
	}
	
	@Override
    public int sizeOfServerUrl(String serverUrl) {
       return this.serviceRepo_byServerURL.sizeOfServerUrl(serverUrl);
    }
	

	@Override
	public List<ServiceInstance> findByServiceId(String serviceId) {
		return this.serviceRepo_byServiceId.findByServiceId(serviceId);
	}


	@Override
	public List<ServiceInstance> getAll() {
		return this.serviceRepo_byServiceId.getAll();
	}





}	
