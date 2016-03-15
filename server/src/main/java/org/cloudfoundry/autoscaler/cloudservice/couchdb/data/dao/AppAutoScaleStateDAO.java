package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.AppAutoScaleState;

public interface AppAutoScaleStateDAO extends CommonDAO {

	public List<AppAutoScaleState> findAll();

	public AppAutoScaleState findByAppId(String appId);

}
