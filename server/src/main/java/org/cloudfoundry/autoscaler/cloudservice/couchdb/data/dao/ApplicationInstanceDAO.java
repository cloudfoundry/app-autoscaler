package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.ApplicationInstance;

public interface ApplicationInstanceDAO extends CommonDAO {

	public List<ApplicationInstance> findAll();

	public List<ApplicationInstance> findByAppId(String appId);

	public List<ApplicationInstance> findByBindId(String bindId);

	public List<ApplicationInstance> findByServiceId(String serviceId);

}
