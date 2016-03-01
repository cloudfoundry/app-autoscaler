package org.cloudfoundry.autoscaler.servicebroker.data.entity.dao.couchdb;

import java.util.List;

import org.cloudfoundry.autoscaler.servicebroker.data.entity.ApplicationInstance;
import org.cloudfoundry.autoscaler.servicebroker.data.entity.dao.ApplicationInstanceDAO;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.GenerateView;
import org.ektorp.support.View;


public class ApplicationInstanceDAOImpl extends CommonDAOImpl implements ApplicationInstanceDAO{

  //  @View ( name = "by_appId", map = "function(doc) { if(doc.type== '' && doc.appId) {emit(doc.appId, doc._id)} }")
	private static class ApplicationInstanceRepository_ByAppId extends TypedCouchdbRepositorySupport<ApplicationInstance>{

		public ApplicationInstanceRepository_ByAppId(CouchDbConnector db) {
			super(ApplicationInstance.class, db, "ApplicationInstance_ByAppId");
			initStandardDesignDocument();
		}

		@GenerateView
	    private List<ApplicationInstance> findByAppId(String appId) {
	    	return  queryView("by_appId", appId);
	    }
	}                                                    
    //@View ( name = "by_bindingId", map = "function(doc) { if(doc.type=='ApplicationInstance_inBroker' && doc.bindingId) {emit(doc.bindingId, doc._id)} }")
	private static class ApplicationInstanceRepository_ByBindingId extends TypedCouchdbRepositorySupport<ApplicationInstance>{

		public ApplicationInstanceRepository_ByBindingId(CouchDbConnector db) {
			super(ApplicationInstance.class, db, "ApplicationInstance_ByBindingId");
			initStandardDesignDocument();
		}

		@GenerateView
		private List<ApplicationInstance> findByBindingId(String bindingId) {
			return queryView("by_bindingId", bindingId);
		}
	}                                         
    //@View ( name = "by_serviceId", map = "function(doc) { if(doc.type=='ApplicationInstance_inBroker' && doc.serviceId) {emit(doc.serviceId, doc._id)} }")
	private static class ApplicationInstanceRepository_ByServiceId extends TypedCouchdbRepositorySupport<ApplicationInstance>{

		public ApplicationInstanceRepository_ByServiceId(CouchDbConnector db) {
			super(ApplicationInstance.class, db, "ApplicationInstance_ByServiceId");
			initStandardDesignDocument();
		}

		@GenerateView
		private List<ApplicationInstance> findByServiceId(String serviceId) {
			return queryView("by_serviceId", serviceId);
		}
		
	}                                        
	
	@View ( name = "without_appId", map = "function(doc) { if(doc.type=='ApplicationInstance_inBroker' && !doc.appId) {emit(doc._id)} }")
	private static class ApplicationInstanceRepository_WithoutAppId extends TypedCouchdbRepositorySupport<ApplicationInstance>{

		public ApplicationInstanceRepository_WithoutAppId(CouchDbConnector db) {
			super(ApplicationInstance.class, db, "ApplicationInstance_WithoutAppId");
			initStandardDesignDocument();
		}

	}	
	
	private	ApplicationInstanceRepository_ByAppId appRepo_byAppId = null;
	private ApplicationInstanceRepository_ByBindingId appRepo_byBindingId = null;
	private	ApplicationInstanceRepository_ByServiceId appRepo_byServiceId = null;

	public ApplicationInstanceDAOImpl(CouchDbConnector db) {
		appRepo_byAppId = new ApplicationInstanceRepository_ByAppId(db);
		appRepo_byBindingId = new ApplicationInstanceRepository_ByBindingId(db);
		appRepo_byServiceId = new ApplicationInstanceRepository_ByServiceId(db);
	}

	@Override
	public <T> TypedCouchdbRepositorySupport<T> getDefaultRepo() {
		return (TypedCouchdbRepositorySupport<T>) this.appRepo_byAppId;
	}

	@Override
	public List<ApplicationInstance> findByAppId(String appId) {
		return this.appRepo_byAppId.findByAppId(appId);
	}
	
	@Override
	public List<ApplicationInstance> findByBindingId(String bindingId) {
		return this.appRepo_byBindingId.findByBindingId(bindingId);
	}

	@Override
	public List<ApplicationInstance> findByServiceId(String serviceId) {
		return this.appRepo_byServiceId.findByServiceId(serviceId);
	}

	@Override
	public List<ApplicationInstance> getAll() {
		return this.appRepo_byAppId.getAll();
	}





}	
