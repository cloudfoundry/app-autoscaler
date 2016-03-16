package org.cloudfoundry.autoscaler.data.couchdb.dao;

import java.util.List;

import org.cloudfoundry.autoscaler.data.couchdb.document.ServiceInstance;

public interface ServiceInstanceDAO extends CommonDAO {

	public List<ServiceInstance> findByServerUrl(String serverUrl);

	public List<ServiceInstance> findByServiceId(String serviceId);

}
