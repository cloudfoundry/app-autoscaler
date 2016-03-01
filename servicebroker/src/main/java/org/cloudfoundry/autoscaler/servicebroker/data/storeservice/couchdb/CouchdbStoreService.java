package org.cloudfoundry.autoscaler.servicebroker.data.storeservice.couchdb;

import java.net.MalformedURLException;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;
import java.util.UUID;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.servicebroker.Constants;
import org.cloudfoundry.autoscaler.servicebroker.data.entity.ApplicationInstance;
import org.cloudfoundry.autoscaler.servicebroker.data.entity.ServiceInstance;
import org.cloudfoundry.autoscaler.servicebroker.data.entity.dao.ApplicationInstanceDAO;
import org.cloudfoundry.autoscaler.servicebroker.data.entity.dao.ServiceInstanceDAO;
import org.cloudfoundry.autoscaler.servicebroker.data.entity.dao.couchdb.ApplicationInstanceDAOImpl;
import org.cloudfoundry.autoscaler.servicebroker.data.entity.dao.couchdb.ServiceInstanceDAOImpl;
import org.cloudfoundry.autoscaler.servicebroker.data.storeservice.IDataStoreService;
import org.cloudfoundry.autoscaler.servicebroker.exception.ProxyInitilizedFailedException;
import org.cloudfoundry.autoscaler.servicebroker.mgr.ConfigManager;
import org.ektorp.CouchDbConnector;
import org.ektorp.CouchDbInstance;
import org.ektorp.http.StdHttpClient;
import org.ektorp.http.StdHttpClient.Builder;
import org.ektorp.impl.StdCouchDbConnector;
import org.ektorp.impl.StdCouchDbInstance;



public class CouchdbStoreService implements IDataStoreService {

	private static final Logger logger = Logger.getLogger(CouchdbStoreService.class);
	private static final CouchdbStoreService instance = new CouchdbStoreService();
    
    private ApplicationInstanceDAO applicationStore;
    private ServiceInstanceDAO serviceStore;
    
    public static CouchdbStoreService getInstance() {
        return instance;
    }
    
	private CouchdbStoreService() {
		try {
			CouchDbConnector couchDBObj = initConnection();
			applicationStore = new ApplicationInstanceDAOImpl (couchDBObj);
			serviceStore = new ServiceInstanceDAOImpl (couchDBObj);
			
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
	}

	
	@Override
	public ServiceInstance createService(String serviceId, String serverUrl, String orgId, String spaceId) {
		ServiceInstance service = new ServiceInstance();
		service.setId(UUID.randomUUID().toString());
		service.setServiceId(serviceId);
    	service.setServerUrl(serverUrl);
    	service.setOrgId(orgId);
    	service.setSpaceId(spaceId);
    	service.setTimestamp(System.currentTimeMillis());
    	serviceStore.add(service);
    	
    	return service;
    }

	@Override
	public ApplicationInstance bindApplication (String appId, String serviceId ,String bindingId) {
		ApplicationInstance application = new ApplicationInstance();
		application.setId(UUID.randomUUID().toString());
		application.setAppId(appId);
		application.setBindingId(bindingId);
		application.setServiceId(serviceId);
		application.setTimestamp(System.currentTimeMillis());
    	applicationStore.add(application);
   		return application;
   	}
	
	@Override
	public void unbindApplication(String bindingId) {

		try {
			List<ApplicationInstance> applications = applicationStore.findByBindingId(bindingId);
			if (applications != null && applications.size() > 0){
				for (ApplicationInstance application : applications){
			    	applicationStore.remove(application);			
				}
			} 
		} catch (org.ektorp.DocumentNotFoundException e) {
		}
		
	}

	@Override
	public void deleteService(String serviceId) {
		try {
			ServiceInstance serviceInstance = getServiceInstanceByServiceId(serviceId);
			if (serviceInstance != null) {
		    	serviceStore.remove(serviceInstance);		
			} 
		} catch (org.ektorp.DocumentNotFoundException e) {
		}
	}

	@Override
	public List<String> getServiceInstanceIdByServerURL(String serverUrl) {
		List<String> serviceIds = null;
		try {
			List<ServiceInstance> services = serviceStore.findByServerURL(serverUrl);
			if (services != null && !services.isEmpty()) {
				serviceIds = new LinkedList<String>();
				for (ServiceInstance service : services) {
					serviceIds.add(service.getServiceId());
				}
			}
		} catch (org.ektorp.DocumentNotFoundException e) {
			//ignore the exception when document not found
		}
		return serviceIds;
	}

	@Override
	public int getWorkloadSummaryByServerURL(String serverUrl) {
		try {
			return serviceStore.sizeOfServerUrl(serverUrl);
		} catch (org.ektorp.DocumentNotFoundException e) {
			//ignore the exception when document not found
		}
		return 0;
	}
	
	@Override
	public List<String> getBoundAppIdByServiceId(String serviceId) {
		
		List<String> appIds = null;
		try {
			List<ApplicationInstance> applications = applicationStore.findByServiceId(serviceId);
			if (applications != null && !applications.isEmpty()){ 
				appIds = new LinkedList<String>();
				for (ApplicationInstance application : applications) {
					appIds.add(application.getAppId());
				}
			}
		} catch (org.ektorp.DocumentNotFoundException e) {
			//ignore the exception when document not found
		}

		return appIds;
	}
	
	@Override
	public ApplicationInstance getBoundAppByBindingId(String bindingId) {
		
		try {
			List<ApplicationInstance> applications = applicationStore.findByBindingId(bindingId);
			if (applications != null && applications.size() > 0) {
				return applications.get(0);  //return the 1st element as bindingId is the unique id. 
			}
		} catch (org.ektorp.DocumentNotFoundException e) {
			//ignore the exception when document not found
		}
		return null;
	}
		

	@Override
	public List<ApplicationInstance> getBoundAppByAppId(String appId) {
		try {
			return applicationStore.findByAppId(appId);
 		} catch (org.ektorp.DocumentNotFoundException e) {
		}
		return null;
	}
	

	@Override
	public ServiceInstance getServiceInstanceByServiceId(String serviceId) {

		try {
			List<ServiceInstance>  serviceInstances = serviceStore.findByServiceId(serviceId);
			if (serviceInstances != null && serviceInstances.size() > 0)
				return serviceInstances.get(0);	 //return the 1st element as serviceId is the unique id. 	
		} catch (org.ektorp.DocumentNotFoundException e) {
		}

		return null;
	
	}	
	
	private CouchDbConnector initConnection() throws MalformedURLException, ProxyInitilizedFailedException {
		
    	String username = ConfigManager.get("couchdbUsername");
    	String password = ConfigManager.getDecryptedString("couchdbPasswordBase64Encoded");
    	String host = ConfigManager.get("couchdbHost");
    	int port = ConfigManager.getInt("couchdbPort");
    	int timeout = ConfigManager.getInt("couchdbTimeout");
    	boolean enableSSL =  ConfigManager.getBoolean("couchdbEnableSSL", false);
    	String dbName = ConfigManager.get("couchdbDBName");
    
		Builder builder = new StdHttpClient.Builder();
		builder = builder
				.host(host)
				.port(port)
				.connectionTimeout(timeout)
				.enableSSL(enableSSL);

		if (username != null && !username.isEmpty() && password != null && !password.isEmpty() ) {
			builder = builder.username(username).password(password);
		}

		CouchDbInstance dbInstance = new StdCouchDbInstance(builder.build());
		CouchDbConnector couchDB = new StdCouchDbConnector(dbName, dbInstance);

		return couchDB;
		
	}



}
