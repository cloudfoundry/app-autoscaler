package org.cloudfoundry.autoscaler.cloudservice.couchdb.connection;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.collection.ApplicationRepositoryCollection;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.collection.BoundAppRepositoryCollection;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.collection.ConfigRepositoryCollection;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.collection.MetricDBSegmentRepositoryCollection;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.collection.ScalingHistoryRepositoryCollection;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.collection.ScalingPolicyRepositoryCollection;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.collection.ScalingStateRepositoryCollection;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.collection.TriggerRecordRepositoryCollection;
import org.ektorp.CouchDbConnector;

public class ServerSeperatedViewRepoConnectionManager extends
TypedRepoConnectionManager {

	private static final Logger logger = Logger
			.getLogger(ServerSeperatedViewRepoConnectionManager.class);
	private TriggerRecordRepositoryCollection triggerRecordRepo;
	private BoundAppRepositoryCollection boundAppRepo;
	private ConfigRepositoryCollection configRepo;
	private MetricDBSegmentRepositoryCollection metricDBSegmentRepo;

	private ScalingPolicyRepositoryCollection scalingPolicyRepo;
	private ApplicationRepositoryCollection applicationRepo;
	private ScalingHistoryRepositoryCollection scalingHistoryRepo;
	private ScalingStateRepositoryCollection scalingStateRepo;
	

	public ServerSeperatedViewRepoConnectionManager(String dbName,
			String userName, String password, String host, int port, boolean enableSSL,
			int timeout) {
		this(dbName, userName, password, host,port, enableSSL, timeout, false);
	}

	public ServerSeperatedViewRepoConnectionManager(String dbName,
			String userName, String password, String host, int port,  boolean enableSSL,
			int timeout, boolean initDesignDocument) {
		try {
			CouchDbConnector couchdb = new CouchDbConnectionManager(dbName,
					userName, password, host, port, enableSSL, timeout).getDb();
			initRepo(couchdb, initDesignDocument);

		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
	}

	

	protected void initRepo(CouchDbConnector db, boolean createIfNotExist) {
		triggerRecordRepo = new TriggerRecordRepositoryCollection(
				db, createIfNotExist);
		boundAppRepo = new BoundAppRepositoryCollection(
				db, createIfNotExist);
		configRepo = new ConfigRepositoryCollection(
				db, createIfNotExist);
		metricDBSegmentRepo = new MetricDBSegmentRepositoryCollection(
				db, createIfNotExist);

		scalingPolicyRepo = new ScalingPolicyRepositoryCollection(
				db, createIfNotExist);
		applicationRepo = new ApplicationRepositoryCollection(
				db, createIfNotExist);
		scalingHistoryRepo = new ScalingHistoryRepositoryCollection(
				db, createIfNotExist);
		scalingStateRepo = new ScalingStateRepositoryCollection(
				db, createIfNotExist);
		
	}

	public TriggerRecordRepositoryCollection getTriggerRecordRepo() {
		return triggerRecordRepo;
	}

	public void setTriggerRecordRepo(
			TriggerRecordRepositoryCollection triggerRecordRepo) {
		this.triggerRecordRepo = triggerRecordRepo;
	}

	public BoundAppRepositoryCollection getBoundAppRepo() {
		return boundAppRepo;
	}

	public void setBoundAppRepo(BoundAppRepositoryCollection boundAppRepo) {
		this.boundAppRepo = boundAppRepo;
	}

	public ConfigRepositoryCollection getConfigRepo() {
		return configRepo;
	}

	public void setConfigRepo(ConfigRepositoryCollection configRepo) {
		this.configRepo = configRepo;
	}

	public MetricDBSegmentRepositoryCollection getMetricDBSegmentRepo() {
		return metricDBSegmentRepo;
	}

	public void setMetricDBSegmentRepo(
			MetricDBSegmentRepositoryCollection metricDBSegmentRepo) {
		this.metricDBSegmentRepo = metricDBSegmentRepo;
	}

	public ScalingPolicyRepositoryCollection getScalingPolicyRepo() {
		return scalingPolicyRepo;
	}

	public void setScalingPolicyRepo(
			ScalingPolicyRepositoryCollection scalingPolicyRepo) {
		this.scalingPolicyRepo = scalingPolicyRepo;
	}

	public ApplicationRepositoryCollection getApplicationRepo() {
		return applicationRepo;
	}

	public void setApplicationRepo(ApplicationRepositoryCollection applicationRepo) {
		this.applicationRepo = applicationRepo;
	}

	public ScalingHistoryRepositoryCollection getScalingHistoryRepo() {
		return scalingHistoryRepo;
	}

	public void setScalingHistoryRepo(
			ScalingHistoryRepositoryCollection scalingHistoryRepo) {
		this.scalingHistoryRepo = scalingHistoryRepo;
	}

	public ScalingStateRepositoryCollection getScalingStateRepo() {
		return scalingStateRepo;
	}

	public void setScalingStateRepo(
			ScalingStateRepositoryCollection scalingStateRepo) {
		this.scalingStateRepo = scalingStateRepo;
	}

	




}
