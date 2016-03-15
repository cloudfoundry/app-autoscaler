package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.AppInstanceMetrics;

public interface AppInstanceMetricsDAO extends CommonDAO {

	public List<AppInstanceMetrics> findAll();

	public List<AppInstanceMetrics> findByAppId(String appId);

	public List<AppInstanceMetrics> findByAppIdBetween(String appId, long startTimestamp, long endTimestamp)
			throws Exception;

	public List<AppInstanceMetrics> findByServiceIdBefore(String serviceId, long olderThan) throws Exception;

	public List<AppInstanceMetrics> findByAppIdAfter(String appId, long timestamp) throws Exception;

}
