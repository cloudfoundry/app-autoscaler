package org.cloudfoundry.autoscaler.cloudservice.couchdb.connection;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.collection.AppInstanceMetricsRepositoryCollection;


public class ServerMetricDBConnectionManager 
{

	private static final Logger logger = Logger.getLogger(ServerMetricDBConnectionManager.class);
	private CouchDbConnectionManager dbConnection; 

	private AppInstanceMetricsRepositoryCollection appInstanceMetricRepo;

	public ServerMetricDBConnectionManager(String dbName, String userName, String password, String host, int port, boolean enableSSL, int timeout) {
		this(dbName, userName, password, host, port, enableSSL, timeout, false);
	}


	public ServerMetricDBConnectionManager(String dbName, String userName, String password, String host, int port, boolean enableSSL, int timeout, boolean initDesignDocument) {
		try
		{
			dbConnection = new CouchDbConnectionManager(dbName,userName, password, host, port, enableSSL, timeout);
			appInstanceMetricRepo = new AppInstanceMetricsRepositoryCollection(dbConnection.getDb() , initDesignDocument);
		}
		catch(Exception e)
		{
			logger.error(e.getMessage(),e);
		}
	}
	

	public AppInstanceMetricsRepositoryCollection getAppInstanceMetricRepo() {
		return appInstanceMetricRepo;
	}


	public void setAppInstanceMetricRepo(
			AppInstanceMetricsRepositoryCollection appInstanceMetricRepo) {
		this.appInstanceMetricRepo = appInstanceMetricRepo;
	}

	public boolean deleteMetricDB(String dbName){
		return dbConnection.deleteDB(dbName);
	}

}
