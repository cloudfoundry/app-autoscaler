package org.cloudfoundry.autoscaler.servicebroker.data.storeservice;

import java.util.List;

import org.cloudfoundry.autoscaler.servicebroker.data.entity.ApplicationInstance;
import org.cloudfoundry.autoscaler.servicebroker.data.entity.ServiceInstance;


public interface IDataStoreService {
	
	public ServiceInstance createService(String serviceId, String serverUrl, String orgId, String spaceId);
	
	public ApplicationInstance bindApplication(String appId, String serviceId, String bindingId);

	public void unbindApplication(String bindingId);

	public void deleteService(String serviceId);

	//service url => multiple service instances
	public List<String> getServiceInstanceIdByServerURL(String serverUrl); 

	// service url => the number of service instances
	public int getWorkloadSummaryByServerURL(String serverUrl); 
	
	//service id => multiple bound app
	public List<String> getBoundAppIdByServiceId(String serviceId);  
	
	//app id => it is possible to have more binding record per app.  But in current implementation, we only allow one app binding one service instance
	public List<ApplicationInstance> getBoundAppByAppId(String appId);  

	//service id => unique id for a service instance 
	public ServiceInstance getServiceInstanceByServiceId(String serviceId); 
	
	//binding id => unique id for a binding
	public ApplicationInstance getBoundAppByBindingId(String bindingId);  

	
}
