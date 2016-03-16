package org.cloudfoundry.autoscaler.data.couchdb.connection.manager;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.data.couchdb.dao.AppInstanceMetricsDAO;
import org.cloudfoundry.autoscaler.data.couchdb.dao.impl.AppInstanceMetricsDAOImpl;

public class ServerMetricDAOManager {

	private static final Logger logger = Logger.getLogger(ServerMetricDAOManager.class);
	private CouchDbConnectionManager dbConnection;

	private AppInstanceMetricsDAO appInstanceMetricDao;

	public ServerMetricDAOManager(String dbName, String userName, String password, String host, int port,
			boolean enableSSL, int timeout) {
		this(dbName, userName, password, host, port, enableSSL, timeout, false);
	}

	public ServerMetricDAOManager(String dbName, String userName, String password, String host, int port,
			boolean enableSSL, int timeout, boolean initDesignDocument) {
		try {
			dbConnection = new CouchDbConnectionManager(dbName, userName, password, host, port, enableSSL, timeout);
			appInstanceMetricDao = new AppInstanceMetricsDAOImpl(dbConnection.getDb(), initDesignDocument);
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
	}

	public AppInstanceMetricsDAO getAppInstanceMetricDao() {
		return appInstanceMetricDao;
	}

	public void setAppInstanceMetricDao(AppInstanceMetricsDAO appInstanceMetricDao) {
		this.appInstanceMetricDao = appInstanceMetricDao;
	}

	public boolean deleteMetricDB(String dbName) {
		return dbConnection.deleteDB(dbName);
	}

}
