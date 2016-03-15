package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao;

import java.util.List;
import java.util.Map;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.TriggerRecord;

public interface TriggerRecordDAO extends CommonDAO {

	public List<TriggerRecord> findAll();

	public List<TriggerRecord> findByAppId(String appId) throws Exception;

	public List<TriggerRecord> findByServerName(String serverName);

	public Map<String, List<TriggerRecord>> getAllTriggers(String serverName) throws Exception;

	public void removeByAppId(String appId) throws Exception;

}
