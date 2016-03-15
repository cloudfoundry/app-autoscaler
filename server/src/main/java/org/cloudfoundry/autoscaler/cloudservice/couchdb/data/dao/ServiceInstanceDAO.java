package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.ServiceInstance;

public interface ServiceInstanceDAO extends CommonDAO {

	public List<ServiceInstance> findByServerUrl(String serverUrl);

	public List<ServiceInstance> findByServiceId(String serviceId);

}
