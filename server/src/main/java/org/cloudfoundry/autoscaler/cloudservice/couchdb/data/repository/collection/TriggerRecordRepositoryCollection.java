package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.collection;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.TriggerRecord;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.TriggerRecordRepository_All;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.TriggerRecordRepository_ByAppId;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.TriggerRecordRepository_ByServerName;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.ektorp.CouchDbConnector;

public class TriggerRecordRepositoryCollection extends
		TypedRepoCollection<TriggerRecord> {

	private static final Logger logger = Logger
			.getLogger(TriggerRecordRepositoryCollection.class);

	private TriggerRecordRepository_All triggerRecordRepo_all;
	private TriggerRecordRepository_ByAppId triggerRecordRepo_byAppId;
	private TriggerRecordRepository_ByServerName triggerRecordRepo_byServerName;

	public TriggerRecordRepositoryCollection(CouchDbConnector db) {
		triggerRecordRepo_all = new TriggerRecordRepository_All(db);
		triggerRecordRepo_byAppId = new TriggerRecordRepository_ByAppId(db);
		triggerRecordRepo_byServerName = new TriggerRecordRepository_ByServerName(
				db);

	}

	public TriggerRecordRepositoryCollection(CouchDbConnector db,
			boolean initDesignDocument) {
		this(db);
		if (initDesignDocument)
			try {
				initAllRepos();
			} catch (Exception e) {
				logger.error(e.getMessage(), e);
			}
	}

	@Override
	public List<TypedCouchDbRepositorySupport> getAllRepos() {
		List<TypedCouchDbRepositorySupport> repoList = new ArrayList<TypedCouchDbRepositorySupport>();

		repoList.add(triggerRecordRepo_all);
		repoList.add(triggerRecordRepo_byAppId);
		repoList.add(triggerRecordRepo_byServerName);
		return repoList;
	}

	@Override
	public TypedCouchDbRepositorySupport getDefaultRepo() {
		return triggerRecordRepo_all;
	}

	public List<TriggerRecord> findByAppId(String appId) throws Exception {
		try {
			return triggerRecordRepo_byAppId.findByAppId(appId);
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

	public List<TriggerRecord> findByServerName(String serverName) {
		try {
			return triggerRecordRepo_byServerName.findByServerName(serverName);
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

	public void removeByAppId(String appId) throws Exception {
		List<TriggerRecord> records = findByAppId(appId);
		for (TriggerRecord record : records)
			remove(record);
	}

	public Map<String, List<TriggerRecord>> getAllTriggers(String serverName)
			throws Exception {
		Map<String, List<TriggerRecord>> triggersMap = new HashMap<String, List<TriggerRecord>>();

		List<TriggerRecord> records = findByServerName(serverName); // will add
																	// this line
																	// later
		for (TriggerRecord record : records) {
			String key = record.getId();
			List<TriggerRecord> recordList = triggersMap.get(key);
			if (recordList == null) {
				recordList = new LinkedList<TriggerRecord>();
				triggersMap.put(key, recordList);
			}
			recordList.add(record);
		}

		return triggersMap;
	}
	
	public List<TriggerRecord> getAllRecords() {
		try {
			return this.triggerRecordRepo_all.getAllRecords();
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}


}
