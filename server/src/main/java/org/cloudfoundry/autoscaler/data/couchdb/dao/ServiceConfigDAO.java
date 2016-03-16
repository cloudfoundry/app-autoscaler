package org.cloudfoundry.autoscaler.data.couchdb.dao;

import java.util.List;

import org.cloudfoundry.autoscaler.data.couchdb.document.ServiceConfig;

public interface ServiceConfigDAO extends CommonDAO {

	public List<ServiceConfig> findAll();

}
