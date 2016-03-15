package org.cloudfoundry.autoscaler.cloudservice.couchdb.connection;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.AppAutoScaleStateDAO;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.ApplicationDAO;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.AutoScalerPolicyDAO;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.BoundAppDAO;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.MetricDBSegmentDAO;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.ScalingHistoryDAO;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.ServiceConfigDAO;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.TriggerRecordDAO;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.couchdb.AppAutoScaleStateDAOImpl;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.couchdb.ApplicationDAOImpl;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.couchdb.AutoScalerPolicyDAOImpl;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.couchdb.BoundAppDAOImpl;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.couchdb.MetricDBSegmentDAOImpl;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.couchdb.ScalingHistoryDAOImpl;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.couchdb.ServiceConfigDAOImpl;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.couchdb.TriggerRecordDAOImpl;
import org.ektorp.CouchDbConnector;

public class ServerViewDAOManager extends TypedRepoConnectionManager {

	private static final Logger logger = Logger.getLogger(ServerViewDAOManager.class);
	private TriggerRecordDAO triggerRecordDao = null;
	private BoundAppDAO boundAppDao = null;
	private ServiceConfigDAO serviceConfigDao = null;
	private MetricDBSegmentDAO metricDBSegmentDao = null;

	private AutoScalerPolicyDAO autoScalerPolicyDao = null;
	private ApplicationDAO applicationDao = null;
	private ScalingHistoryDAO scalingHistoryDao = null;
	private AppAutoScaleStateDAO appAutoScalerStateDao = null;

	public ServerViewDAOManager(String dbName, String userName, String password, String host,
			int port, boolean enableSSL, int timeout) {
		this(dbName, userName, password, host, port, enableSSL, timeout, false);
	}

	public ServerViewDAOManager(String dbName, String userName, String password, String host,
			int port, boolean enableSSL, int timeout, boolean initDesignDocument) {
		try {
			CouchDbConnector couchdb = new CouchDbConnectionManager(dbName, userName, password, host, port, enableSSL,
					timeout).getDb();
			initRepo(couchdb, initDesignDocument);

		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
	}

	protected void initRepo(CouchDbConnector db, boolean createIfNotExist) {

		triggerRecordDao = new TriggerRecordDAOImpl(db, createIfNotExist);
		boundAppDao = new BoundAppDAOImpl(db, createIfNotExist);
		serviceConfigDao = new ServiceConfigDAOImpl(db, createIfNotExist);
		metricDBSegmentDao = new MetricDBSegmentDAOImpl(db, createIfNotExist);
		autoScalerPolicyDao = new AutoScalerPolicyDAOImpl(db, createIfNotExist);
		applicationDao = new ApplicationDAOImpl(db, createIfNotExist);
		scalingHistoryDao = new ScalingHistoryDAOImpl(db, createIfNotExist);
		appAutoScalerStateDao = new AppAutoScaleStateDAOImpl(db, createIfNotExist);

	}

	public TriggerRecordDAO getTriggerRecordDao() {
		return triggerRecordDao;
	}

	public void setTriggerRecordDao(TriggerRecordDAO triggerRecordDao) {
		this.triggerRecordDao = triggerRecordDao;
	}

	public BoundAppDAO getBoundAppDao() {
		return boundAppDao;
	}

	public void setBoundAppDao(BoundAppDAO boundAppDao) {
		this.boundAppDao = boundAppDao;
	}

	public ServiceConfigDAO getServiceConfigDao() {
		return serviceConfigDao;
	}

	public void setServiceConfigDao(ServiceConfigDAO serviceConfigDao) {
		this.serviceConfigDao = serviceConfigDao;
	}

	public MetricDBSegmentDAO getMetricDBSegmentDao() {
		return metricDBSegmentDao;
	}

	public void setMetricDBSegmentDao(MetricDBSegmentDAO metricDBSegmentDao) {
		this.metricDBSegmentDao = metricDBSegmentDao;
	}

	public AutoScalerPolicyDAO getAutoScalerPolicyDao() {
		return autoScalerPolicyDao;
	}

	public void setAutoScalerPolicyDao(AutoScalerPolicyDAO autoScalerPolicyDao) {
		this.autoScalerPolicyDao = autoScalerPolicyDao;
	}

	public ApplicationDAO getApplicationDao() {
		return applicationDao;
	}

	public void setApplicationDao(ApplicationDAO applicationDao) {
		this.applicationDao = applicationDao;
	}

	public ScalingHistoryDAO getScalingHistoryDao() {
		return scalingHistoryDao;
	}

	public void setScalingHistoryDao(ScalingHistoryDAO scalingHistoryDao) {
		this.scalingHistoryDao = scalingHistoryDao;
	}

	public AppAutoScaleStateDAO getAppAutoScalerStateDao() {
		return appAutoScalerStateDao;
	}

	public void setAppAutoScalerStateDao(AppAutoScaleStateDAO appAutoScalerStateDao) {
		this.appAutoScalerStateDao = appAutoScalerStateDao;
	}

}
