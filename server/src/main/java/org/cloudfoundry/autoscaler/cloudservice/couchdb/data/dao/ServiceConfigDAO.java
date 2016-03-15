package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.ServiceConfig;

public interface ServiceConfigDAO extends CommonDAO {

	public List<ServiceConfig> findAll();

}
