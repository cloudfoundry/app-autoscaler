package org.cloudfoundry.autoscaler.data.couchdb.dao;

import java.util.List;

import org.cloudfoundry.autoscaler.data.couchdb.document.AppAutoScaleState;

public interface AppAutoScaleStateDAO extends CommonDAO {

	public List<AppAutoScaleState> findAll();

	public AppAutoScaleState findByAppId(String appId);

}
