package org.cloudfoundry.autoscaler.servicebroker.data.entity.dao;

import java.util.List;

import org.cloudfoundry.autoscaler.servicebroker.data.entity.ApplicationInstance;


public interface ApplicationInstanceDAO extends CommonDAO{

	public List<ApplicationInstance> findByAppId(String appId);
	
	public List<ApplicationInstance> findByBindingId(String bindingId);
	
	public List<ApplicationInstance> findByServiceId(String serviceId);
	
	public List<ApplicationInstance> getAll();

}	
