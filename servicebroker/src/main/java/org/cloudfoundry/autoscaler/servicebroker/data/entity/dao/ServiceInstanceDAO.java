package org.cloudfoundry.autoscaler.servicebroker.data.entity.dao;

import java.util.List;

import org.cloudfoundry.autoscaler.servicebroker.data.entity.ServiceInstance;


public interface ServiceInstanceDAO extends CommonDAO {

	public List<ServiceInstance> findByServiceId(String serviceId);
	
	public List<ServiceInstance> findByServerURL(String serverURL);
	
	public int sizeOfServerUrl(String serverURL);
	
	public List<ServiceInstance> getAll();
	
}	
