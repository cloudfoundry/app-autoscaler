package org.cloudfoundry.autoscaler.data.couchdb.dao;

import java.util.List;

import org.cloudfoundry.autoscaler.data.couchdb.document.ApplicationInstance;

public interface ApplicationInstanceDAO extends CommonDAO {

	public List<ApplicationInstance> findAll();

	public List<ApplicationInstance> findByAppId(String appId);

	public List<ApplicationInstance> findByBindId(String bindId);

	public List<ApplicationInstance> findByServiceId(String serviceId);

}
