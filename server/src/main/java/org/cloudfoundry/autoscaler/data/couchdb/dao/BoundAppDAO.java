package org.cloudfoundry.autoscaler.data.couchdb.dao;

import java.util.List;

import org.cloudfoundry.autoscaler.data.couchdb.document.BoundApp;

public interface BoundAppDAO extends CommonDAO {

	public List<BoundApp> findAll();

	public BoundApp findByAppId(String appId) throws Exception;

	public List<BoundApp> findByServerName(String serverName);

	public List<BoundApp> findByServiceIdAndAppId(String serviceId, String appId) throws Exception;

	public List<BoundApp> findByServiceId(String serviceId) throws Exception;

	public List<BoundApp> getAllBoundApps(String serverName);

	public void removeByServiceIdAndAppId(String serviceId, String appId) throws Exception;

	public void updateByServiceIdAndAppId(String serviceId, String appId, String appType, String appName,
			String serverName, boolean insertIfNotFound) throws Exception;

}
