package org.cloudfoundry.autoscaler.data.couchdb.dao.impl;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.data.couchdb.dao.TriggerRecordDAO;
import org.cloudfoundry.autoscaler.data.couchdb.dao.base.TypedCouchDbRepositorySupport;
import org.cloudfoundry.autoscaler.data.couchdb.document.TriggerRecord;
import org.ektorp.ComplexKey;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

public class TriggerRecordDAOImpl extends CommonDAOImpl implements TriggerRecordDAO {

	@View(name = "byAll", map = "function(doc) { if (doc.type == 'TriggerRecord' ) emit( [doc.appId, doc.serverName], doc._id )}")
	public class TriggerRecordRepository_All extends TypedCouchDbRepositorySupport<TriggerRecord> {

		public TriggerRecordRepository_All(CouchDbConnector db) {
			super(TriggerRecord.class, db, "TriggerRecord_byAll");
		}

		public List<TriggerRecord> getAllRecords() {
			return queryView("byAll");
		}

	}

	@View(name = "by_appId", map = "function(doc) { if (doc.type=='TriggerRecord' && doc.appId) { emit([ doc.appId], doc._id) } }")
	public class TriggerRecordRepository_ByAppId extends TypedCouchDbRepositorySupport<TriggerRecord> {

		public TriggerRecordRepository_ByAppId(CouchDbConnector db) {
			super(TriggerRecord.class, db, "TriggerRecord_ByAppId");
		}

		public List<TriggerRecord> findByAppId(String appId) throws Exception {
			ComplexKey key = ComplexKey.of(appId);
			return queryView("by_appId", key);
		}
	}

	@View(name = "by_servername", map = "function(doc) { if (doc.type=='TriggerRecord' && doc.serverName) { emit([doc.serverName], doc._id) } }")
	public class TriggerRecordRepository_ByServerName extends TypedCouchDbRepositorySupport<TriggerRecord> {

		public TriggerRecordRepository_ByServerName(CouchDbConnector db) {
			super(TriggerRecord.class, db, "TriggerRecord_ByServerName");
		}

		public List<TriggerRecord> findByServerName(String serverName) {
			ComplexKey key = ComplexKey.of(serverName);
			return queryView("by_servername", key);
		}

	}

	private static final Logger logger = Logger.getLogger(TriggerRecordDAOImpl.class);
	private TriggerRecordRepository_All triggerRecordAllRepo = null;
	private TriggerRecordRepository_ByAppId triggerRecordByAppIdRepo = null;
	private TriggerRecordRepository_ByServerName triggerRecordByServerNameRepo = null;

	public TriggerRecordDAOImpl(CouchDbConnector db) {
		this.triggerRecordAllRepo = new TriggerRecordRepository_All(db);
		this.triggerRecordByAppIdRepo = new TriggerRecordRepository_ByAppId(db);
		this.triggerRecordByServerNameRepo = new TriggerRecordRepository_ByServerName(db);
	}

	public TriggerRecordDAOImpl(CouchDbConnector db, boolean initDesignDocument) {
		this(db);
		if (initDesignDocument) {
			try {
				initAllRepos();
			} catch (Exception e) {
				logger.error(e.getMessage(), e);
			}
		}

	}

	@Override
	public List<TriggerRecord> findAll() {
		// TODO Auto-generated method stub
		return this.triggerRecordAllRepo.getAllRecords();
	}

	@Override
	public List<TriggerRecord> findByAppId(String appId) throws Exception {
		// TODO Auto-generated method stub
		return this.triggerRecordByAppIdRepo.findByAppId(appId);
	}

	@Override
	public List<TriggerRecord> findByServerName(String serverName) {
		// TODO Auto-generated method stub
		return this.triggerRecordByServerNameRepo.findByServerName(serverName);
	}

	@SuppressWarnings("unchecked")
	@Override
	public <T> TypedCouchDbRepositorySupport<T> getDefaultRepo() {
		// TODO Auto-generated method stub
		return (TypedCouchDbRepositorySupport<T>) this.triggerRecordAllRepo;
	}

	@Override
	public Map<String, List<TriggerRecord>> getAllTriggers(String serverName) throws Exception {
		Map<String, List<TriggerRecord>> triggersMap = new HashMap<String, List<TriggerRecord>>();
		List<TriggerRecord> records = findByServerName(serverName);
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

	@Override
	public void removeByAppId(String appId) throws Exception {
		List<TriggerRecord> records = findByAppId(appId);
		for (TriggerRecord record : records)
			remove(record);
	}

	@SuppressWarnings("unchecked")
	@Override
	public <T> List<TypedCouchDbRepositorySupport<T>> getAllRepos() {
		// TODO Auto-generated method stub
		List<TypedCouchDbRepositorySupport<T>> repoList = new ArrayList<TypedCouchDbRepositorySupport<T>>();
		repoList.add((TypedCouchDbRepositorySupport<T>) this.triggerRecordAllRepo);
		repoList.add((TypedCouchDbRepositorySupport<T>) this.triggerRecordByAppIdRepo);
		repoList.add((TypedCouchDbRepositorySupport<T>) this.triggerRecordByServerNameRepo);
		return repoList;
	}

}
