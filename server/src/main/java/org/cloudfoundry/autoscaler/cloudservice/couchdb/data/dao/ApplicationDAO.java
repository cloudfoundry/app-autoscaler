package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.Application;

public interface ApplicationDAO extends CommonDAO {

	public List<Application> findAll();

	public Application findByAppId(String appId);

	public Application findByBindId(String bindId);

	public List<Application> findByPolicyId(String policyId);

	public List<Application> findByServiceIdState(String serviceId);

	public List<Application> findByServiceId(String serviceId);

	public List<Application> findByServiceIdAndState(String serviceId);

}
