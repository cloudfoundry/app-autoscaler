package org.cloudfoundry.autoscaler.data.couchdb;

import java.util.ArrayList;
import java.util.Calendar;
import java.util.Collections;
import java.util.Comparator;
import java.util.HashMap;
import java.util.Iterator;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;
import java.util.Set;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.bean.Trigger;
import org.cloudfoundry.autoscaler.common.util.ConfigManager;
import org.cloudfoundry.autoscaler.data.AutoScalingDataStore;
import org.cloudfoundry.autoscaler.data.couchdb.connection.manager.ServerDAOManager;
import org.cloudfoundry.autoscaler.data.couchdb.connection.manager.ServerMetricDAOManager;
import org.cloudfoundry.autoscaler.data.couchdb.dao.AppAutoScaleStateDAO;
import org.cloudfoundry.autoscaler.data.couchdb.dao.AppInstanceMetricsDAO;
import org.cloudfoundry.autoscaler.data.couchdb.dao.ApplicationDAO;
import org.cloudfoundry.autoscaler.data.couchdb.dao.AutoScalerPolicyDAO;
import org.cloudfoundry.autoscaler.data.couchdb.dao.BoundAppDAO;
import org.cloudfoundry.autoscaler.data.couchdb.dao.MetricDBSegmentDAO;
import org.cloudfoundry.autoscaler.data.couchdb.dao.ScalingHistoryDAO;
import org.cloudfoundry.autoscaler.data.couchdb.dao.ServiceConfigDAO;
import org.cloudfoundry.autoscaler.data.couchdb.dao.TriggerRecordDAO;
import org.cloudfoundry.autoscaler.data.couchdb.document.AppAutoScaleState;
import org.cloudfoundry.autoscaler.data.couchdb.document.AppInstanceMetrics;
import org.cloudfoundry.autoscaler.data.couchdb.document.Application;
import org.cloudfoundry.autoscaler.data.couchdb.document.AutoScalerPolicy;
import org.cloudfoundry.autoscaler.data.couchdb.document.BoundApp;
import org.cloudfoundry.autoscaler.data.couchdb.document.MetricDBSegment;
import org.cloudfoundry.autoscaler.data.couchdb.document.ScalingHistory;
import org.cloudfoundry.autoscaler.data.couchdb.document.ServiceConfig;
import org.cloudfoundry.autoscaler.data.couchdb.document.TriggerRecord;
import org.cloudfoundry.autoscaler.exceptions.DataStoreException;
import org.cloudfoundry.autoscaler.exceptions.PolicyNotFoundException;
import org.cloudfoundry.autoscaler.manager.ScalingHistoryFilter;
import org.cloudfoundry.autoscaler.util.AutoScalerEnvUtil;

import com.fasterxml.jackson.databind.ObjectMapper;

public class CouchdbStorageService implements AutoScalingDataStore {
	private static final Logger logger = Logger.getLogger(CouchdbStorageService.class);

	private static final String username = ConfigManager.get("couchdbUsername");
	private static final String password = ConfigManager.get("couchdbPassword");
	private static final String host = ConfigManager.get("couchdbHost");
	private static final int port = ConfigManager.getInt("couchdbPort");
	private static final boolean enableSSL = false;
	private static final int timeout = ConfigManager.getInt("couchdbTimeout");

	private static final String DB_NAME = ConfigManager.get("couchdbDBName");
	private static final String metricDBPrefix = ConfigManager.get("couchdbMetricDBPrefix") + "-";
	private static final boolean initDesignDocument = ConfigManager.getBoolean("couchdbDBInitDesignDocument", true);
	private static final long metricDBStaleTime = ConfigManager.getInt("couchdbMetricDBStaleAfter") * 1000 * 60L;
	private static final ObjectMapper mapper = new ObjectMapper();
	private static final String serverName = AutoScalerEnvUtil.getServerName();

	private TriggerRecordDAO triggerRecordDao;
	private BoundAppDAO boundAppDao;
	private ServiceConfigDAO serviceConfigDao;
	private MetricDBSegmentDAO metricDBSegmentDao;
	private AutoScalerPolicyDAO autoScalerPolicyDao;
	private ApplicationDAO applicationDao;
	private AppAutoScaleStateDAO appAutoScaleStateDao;
	private ScalingHistoryDAO scalingHistoryDao;

	private List<AppInstanceMetricsDAO> appInstanceMetricsDAOList;
	private List<MetricDBSegment> metricDBSegmentList;
	private int curMetricDBSegmentAnchor;

	private static volatile CouchdbStorageService instance;

	private CouchdbStorageService() {
		initCouchdb();
	}

	public static CouchdbStorageService getInstance() {
		if (instance == null) {
			synchronized (CouchdbStorageService.class) {
				if (instance == null)
					instance = new CouchdbStorageService();
			}
		}
		return instance;
	}

	private void initCouchdb() {

		ServerDAOManager ScalingRepoManager = null;

		ScalingRepoManager = new ServerDAOManager(DB_NAME, username, password, host, port,
				enableSSL, timeout, initDesignDocument);
		triggerRecordDao = ScalingRepoManager.getTriggerRecordDao();
		boundAppDao = ScalingRepoManager.getBoundAppDao();
		serviceConfigDao = ScalingRepoManager.getServiceConfigDao();
		metricDBSegmentDao = ScalingRepoManager.getMetricDBSegmentDao();
		autoScalerPolicyDao = ScalingRepoManager.getAutoScalerPolicyDao();
		applicationDao = ScalingRepoManager.getApplicationDao();
		scalingHistoryDao = ScalingRepoManager.getScalingHistoryDao();
		appAutoScaleStateDao = ScalingRepoManager.getAppAutoScalerStateDao();

		initExistingMetricDB();

	}

	@SuppressWarnings("unchecked")
	public void initExistingMetricDB() {

		Calendar now = Calendar.getInstance();

		appInstanceMetricsDAOList = new ArrayList<AppInstanceMetricsDAO>();
		metricDBSegmentList = metricDBSegmentDao.findLastestMetricDBs(serverName);
		MetricDBSegment curSegment = MetricDBSegmentManager.getInstance().getMetricDBSegment(now, serverName);

		if (metricDBSegmentList != null) {
			Collections.sort(metricDBSegmentList);

			// stale the old db segment records & rollout old database
			Iterator<MetricDBSegment> iter = metricDBSegmentList.iterator();
			while (iter.hasNext()) {
				MetricDBSegment seg = iter.next();
				if (seg.getEndTimestamp() != Long.MAX_VALUE
						&& seg.getEndTimestamp() + metricDBStaleTime < curSegment.getStartTimestamp()) {
					String staledMetricDBName = metricDBPrefix + seg.getMetricDBPostfix();
					logger.info("Remove staled metricDB : " + metricDBPrefix + seg.toString());
					new ServerMetricDAOManager(staledMetricDBName, username, password, host, port, enableSSL,
							timeout).deleteMetricDB(staledMetricDBName);
					metricDBSegmentDao.remove(seg);
					iter.remove();
				}
			}
		} else
			metricDBSegmentList = new ArrayList<MetricDBSegment>();

		boolean createNewSegment = false;
		if (metricDBSegmentList == null || metricDBSegmentList.size() == 0) {
			curSegment.setSegmentSeq(0);
			createNewSegment = true;

		} else {
			MetricDBSegment lastestPreSegment = metricDBSegmentList.get(metricDBSegmentList.size() - 1);
			if (!(curSegment.getMetricDBPostfix().equalsIgnoreCase(lastestPreSegment.getMetricDBPostfix())
					&& curSegment.getEndTimestamp() == lastestPreSegment.getEndTimestamp())) {
				curSegment.setSegmentSeq(lastestPreSegment.getSegmentSeq() + 1);
				createNewSegment = true;

				// cut off preSegment endTimestamp record to reflect the actual
				// end time.
				if (lastestPreSegment.getEndTimestamp() > curSegment.getStartTimestamp()) {
					lastestPreSegment.setEndTimestamp(curSegment.getStartTimestamp() - 1);
					metricDBSegmentDao.updateMetricDBSegment(lastestPreSegment);
				}
			}
		}

		if (createNewSegment) {
			metricDBSegmentList.add(curSegment);
			metricDBSegmentDao.add(curSegment);
		}

		for (MetricDBSegment seg : metricDBSegmentList) {
			String metricDBName = metricDBPrefix + seg.getMetricDBPostfix();
			ServerMetricDAOManager manager = null;

			manager = new ServerMetricDAOManager(metricDBName, username, password, host, port, enableSSL,
					timeout, true);

			AppInstanceMetricsDAO appInstanceMetricsDao = (manager).getAppInstanceMetricDao();
			appInstanceMetricsDAOList.add(appInstanceMetricsDao);
		}

		curMetricDBSegmentAnchor = metricDBSegmentList.size() - 1;

		try {
			logger.info("Init with metricDBSegment : " + mapper.writeValueAsString(metricDBSegmentList));
		} catch (Exception e) {
			logger.error("Error: " + e.getMessage(), e);
		}

	}

	public synchronized boolean addMetricDB(long startTimestamp, int seq) throws Exception {

		if (metricDBSegmentDao.findByServerNameSegmentSeq(serverName, seq) != null) {
			logger.info("Required metric DB is added at timestamp " + startTimestamp + " with seq " + seq);
			return true;
		}

		Calendar now = Calendar.getInstance();
		now.setTimeInMillis(startTimestamp);

		MetricDBSegment newSegment = MetricDBSegmentManager.getInstance().getMetricDBSegment(now, serverName);
		newSegment.setSegmentSeq(seq);
		metricDBSegmentDao.add(newSegment);

		String newMetricDBName = metricDBPrefix + newSegment.getMetricDBPostfix();
		AppInstanceMetricsDAO newAppInstanceMetricsDao = null;
		try {
			newAppInstanceMetricsDao = (new ServerMetricDAOManager(newMetricDBName, username, password, host,
					port, enableSSL, timeout, true)).getAppInstanceMetricDao();

		} catch (Exception e) {
			logger.error("Fail to add new metric DB " + newMetricDBName + " with Error: " + e.getMessage(), e);
			return false;
		}

		appInstanceMetricsDAOList.add(newAppInstanceMetricsDao);
		metricDBSegmentList.add(newSegment);
		curMetricDBSegmentAnchor = metricDBSegmentList.size() - 1;
		logger.info(
				"Add a new metric DB. Current metricDBSegment is : " + mapper.writeValueAsString(metricDBSegmentList));

		return true;

	}

	@Override
	public void addTrigger(Trigger t) throws Exception {
		TriggerRecord triggerRecord = new TriggerRecord(t.getAppName(), t);
		TriggerRecord existingRecord = null;
		existingRecord = (TriggerRecord) triggerRecordDao.tryGet(triggerRecord.getId());
		if (existingRecord != null && existingRecord.getAppId() != null) {
			triggerRecordDao.remove(existingRecord);
		}

		triggerRecord.setServerName(AutoScalerEnvUtil.getServerName());
		triggerRecordDao.add(triggerRecord);
	}

	@Override
	public void removeTrigger(String appId) throws Exception {
		triggerRecordDao.removeByAppId(appId);

	}

	@Override
	public Map<String, List<TriggerRecord>> getAllTriggers() throws Exception {
		return triggerRecordDao.getAllTriggers(serverName);
	}

	@Override
	public void addAppStats(AppInstanceMetrics appInstanceMetrics) throws Exception {
		MetricDBSegment activeMetricDBSegment = metricDBSegmentList.get(curMetricDBSegmentAnchor);
		AppInstanceMetricsDAO activeAppInstanceMetricsDao = appInstanceMetricsDAOList.get(curMetricDBSegmentAnchor);

		long currentTimestamp = appInstanceMetrics.getTimestamp();
		long endTimestamp = activeMetricDBSegment.getEndTimestamp();

		if (currentTimestamp > endTimestamp) {
			// if a new metric db is added and connected, add to the new metric
			// db,
			// otherwise reusing previous ones and retry the add metric db when
			// next metric data comes.
			if (addMetricDB(currentTimestamp, activeMetricDBSegment.getSegmentSeq() + 1))
				activeAppInstanceMetricsDao = appInstanceMetricsDAOList.get(curMetricDBSegmentAnchor);
		}

		activeAppInstanceMetricsDao.add(appInstanceMetrics);
	}

	@Override
	public List<AppInstanceMetrics> getAppStatsHistoryByAppIdAfter(String appId, long newerThan) throws Exception {

		List<AppInstanceMetrics> results = new ArrayList<AppInstanceMetrics>();

		int startingAnchor = 0;
		for (int i = curMetricDBSegmentAnchor; i >= 0; i--) {
			if (newerThan > metricDBSegmentList.get(i).getStartTimestamp()) {
				startingAnchor = i;
				break;
			}
		}
		for (int i = startingAnchor; i <= curMetricDBSegmentAnchor; i++) {
			AppInstanceMetricsDAO appInstanceMetricsDao = appInstanceMetricsDAOList.get(i);
			List<AppInstanceMetrics> appInstanceMetrics = appInstanceMetricsDao.findByAppIdAfter(appId, newerThan);
			if (appInstanceMetrics != null)
				results.addAll(appInstanceMetrics);
		}

		return results;
	}

	@Override
	public ServiceConfig getConfig(String serviceId) throws Exception {
		ServiceConfig config = null;
		config = (ServiceConfig) serviceConfigDao.get(serviceId);
		return config;
	}

	@Override
	public long getSmallestPersistTime() {
		long smallestPersistTime = 0;
		try {
			List<ServiceConfig> configs = getAllServiceConfigs();
			if (configs != null && configs.size() > 0) {
				Collections.sort(configs, new Comparator<ServiceConfig>() {

					@Override
					public int compare(ServiceConfig o1, ServiceConfig o2) {
						long t1 = o1.getPersistTimeInDb();
						long t2 = o2.getPersistTimeInDb();
						if (t1 < t2) {
							return -1;
						} else if (t1 > t2) {
							return 1;
						}
						return 0;
					}

				});
				return configs.get(0).getPersistTimeInDb();
			}
		} catch (Exception e) {
			logger.error(e);
		}
		return smallestPersistTime;
	}

	@Override
	public List<ServiceConfig> getAllServiceConfigs() throws Exception {
		return serviceConfigDao.findAll();
	}

	@Override
	public void addBinding(String serviceId, String appId, String appType, String appName) throws Exception {
		boundAppDao.updateByServiceIdAndAppId(serviceId, appId, appType, appName, serverName, true);
	}

	@Override
	public void updateBinding(String serviceId, String appId, String appType, String appName) throws Exception {
		boundAppDao.updateByServiceIdAndAppId(serviceId, appId, appType, appName, serverName, false);
	}

	@Override
	public void removeBinding(String serviceId, String appId) throws Exception {
		boundAppDao.removeByServiceIdAndAppId(serviceId, appId);
	}

	@Override
	public Map<String, List<BoundApp>> getAllBindings() throws Exception {
		Map<String, List<BoundApp>> bindingsMap = new HashMap<String, List<BoundApp>>();

		List<BoundApp> boundApps = boundAppDao.getAllBoundApps(serverName);

		for (BoundApp boundApp : boundApps) {
			String serviceId = boundApp.getServiceId();
			List<BoundApp> apps = bindingsMap.get(serviceId);
			if (apps == null) {
				apps = new LinkedList<BoundApp>();
				bindingsMap.put(serviceId, apps);
			}
			apps.add(boundApp);
		}
		return bindingsMap;
	}

	@Override
	public Set<String> getAllBoundServiceIds() throws Exception {
		return getAllBindings().keySet();
	}

	@Override
	public List<BoundApp> getAllBindingsByServiceId(String serviceId) throws Exception {
		return boundAppDao.findByServiceId(serviceId);
	}

	@Override
	public String getAppTypeById(String appId) throws Exception {
		BoundApp boundApp = boundAppDao.findByAppId(appId);
		if (boundApp != null) {
			return boundApp.getAppType();
		}
		return "";
	}

	@Override
	public void removeAppStatsWithHistory(String appId) throws Exception {
		// TODO Auto-generated method stub

	}

	@Override
	public void saveApplication(Application app) throws DataStoreException {
		Application existingApp = getApplication(app.getAppId());
		if (existingApp == null)
			applicationDao.add(app);
		else {
			if (app.getId() == null)
				app.setId(existingApp.getId());
			if (app.getRevision() == null) {
				app.setRevision(existingApp.getRevision());
			}
			applicationDao.update(app);
		}
	}

	@Override
	public void removeApplication(String appId) {
		Application app = getApplication(appId);
		applicationDao.remove(app);

	}

	@Override
	public Application getApplicationByBindingId(String bindingId) {
		return applicationDao.findByBindId(bindingId);
	}

	@Override
	public Application getApplication(String appId) {
		return applicationDao.findByAppId(appId);

	}

	@Override
	public void removeApplicationByBindingId(String bindingId) throws DataStoreException {
		Application app = getApplicationByBindingId(bindingId);
		applicationDao.remove(app);
	}

	@Override
	public AutoScalerPolicy getPolicyById(String policyId) throws PolicyNotFoundException {
		return autoScalerPolicyDao.findByPolicyId(policyId);
	}

	@Override
	public String savePolicy(AutoScalerPolicy policy) throws DataStoreException {
		String policyId = policy.getPolicyId();
		if (policyId == null) {
			String uuid = java.util.UUID.randomUUID().toString();
			policy.setPolicyId(uuid);
			policy.setId(uuid);
			autoScalerPolicyDao.add(policy);
		} else {
			AutoScalerPolicy oldPolicy;
			try {
				oldPolicy = getPolicyById(policy.getPolicyId());
			} catch (PolicyNotFoundException e) {
				throw new DataStoreException("Th policy" + policyId + "is not found.", e);
			}
			if (policy.getId() == null)
				policy.setId(policy.getPolicyId());
			if (policy.getRevision() == null) {
				policy.setRevision(oldPolicy.getRevision());
			}
			autoScalerPolicyDao.update(policy);
		}
		return policy.getPolicyId();
	}

	@Override
	public void deletePolicy(String policyId) throws DataStoreException, PolicyNotFoundException {
		AutoScalerPolicy policy = this.getPolicyById(policyId);
		autoScalerPolicyDao.remove(policy);

	}

	@Override
	public void saveScalingState(AppAutoScaleState state) throws DataStoreException {
		try {
			AppAutoScaleState existingState = getScalingState(state.getAppId());
			if (existingState == null)
				appAutoScaleStateDao.add(state);
			else {
				if (state.getId() == null)
					state.setId(existingState.getId());
				if (state.getRevision() == null)
					state.setRevision(existingState.getRevision());
				appAutoScaleStateDao.update(state);
			}
		} catch (org.ektorp.DbAccessException e) {
			throw new DataStoreException(e);
		}

	}

	@Override
	public AppAutoScaleState getScalingState(String appId) {
		return appAutoScaleStateDao.findByAppId(appId);
	}

	@Override
	public List<Application> getApplications(String serviceId) {
		return applicationDao.findByServiceIdAndState(serviceId);
	}

	@Override
	public ScalingHistory getHistoryById(String id) throws DataStoreException {
		if (id == null) {
			return null;
		}
		ScalingHistory history = null;
		try {
			history = (ScalingHistory) scalingHistoryDao.tryGet(id);
		} catch (org.ektorp.DocumentNotFoundException ex) {
		}
		return history;
	}

	@Override
	public void saveScalingHistory(ScalingHistory scalingHistory) throws DataStoreException {
		try {
			ScalingHistory existingHistory = getHistoryById(scalingHistory.getId());
			if (existingHistory == null)
				scalingHistoryDao.add(scalingHistory);
			else {
				if (scalingHistory.getId() == null)
					scalingHistory.setId(existingHistory.getId());
				if (scalingHistory.getRevision() == null)
					scalingHistory.setRevision(existingHistory.getRevision());
				scalingHistoryDao.update(scalingHistory);
			}
		} catch (org.ektorp.DbAccessException e) {
			throw new DataStoreException(e);
		}
	}

	@Override
	public List<ScalingHistory> getHistoryList(ScalingHistoryFilter filter) throws DataStoreException {
		List<ScalingHistory> historyList = scalingHistoryDao.findByScalingTime(filter.getAppId(), filter.getStartTime(),
				filter.getEndTime());
		if (filter.getScaleType() != null) {
			historyList = filterScalingHistoryByScaleType(historyList, filter.getScaleType());
		}
		if (filter.getStatus() != null) {
			historyList = filterScalingHistoryByStatus(historyList, filter.getStatus());
		}
		return historyList;
	}

	private List<ScalingHistory> filterScalingHistoryByScaleType(List<ScalingHistory> historyList, String scaleType) {
		List<ScalingHistory> newList = new ArrayList<ScalingHistory>();
		if (historyList == null)
			return newList;
		for (ScalingHistory history : historyList) {
			if (ScalingHistoryFilter.SCALE_IN_TYPE.equals(scaleType) && history.getAdjustment() < 0)
				newList.add(history);
			else if (ScalingHistoryFilter.SCALE_OUT_TYPE.equals(scaleType) && history.getAdjustment() > 0)
				newList.add(history);
		}
		return newList;
	}

	private List<ScalingHistory> filterScalingHistoryByStatus(List<ScalingHistory> historyList, String status) {
		List<ScalingHistory> newList = new ArrayList<ScalingHistory>();
		if (historyList == null)
			return newList;
		for (ScalingHistory history : historyList) {
			if (history.getStatus() == Integer.parseInt(status))
				newList.add(history);
		}
		return newList;
	}

	@Override
	public int getHistoryCount(ScalingHistoryFilter filter) throws DataStoreException {
		List<ScalingHistory> historyList = scalingHistoryDao.findByScalingTime(filter.getAppId(), filter.getStartTime(),
				filter.getEndTime());
		if (filter.getScaleType() != null) {
			historyList = filterScalingHistoryByScaleType(historyList, filter.getScaleType());
		}
		if (filter.getStatus() != null) {
			historyList = filterScalingHistoryByStatus(historyList, filter.getStatus());
		}
		return historyList.size();
	}

	@Override
	public List<Application> getApplicationsByPolicyId(String policyId) {
		return applicationDao.findByPolicyId(policyId);
	}

	@Override
	public List<AutoScalerPolicy> getAutoScalerPolicies() {
		return autoScalerPolicyDao.getAllRecords();
	}

}
