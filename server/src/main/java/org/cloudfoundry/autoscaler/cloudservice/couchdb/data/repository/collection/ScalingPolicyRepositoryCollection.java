package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.collection;

import java.util.ArrayList;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.AutoScalerPolicy;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.ScalingPolicyRepository_All;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.seperatedViews.ScalingPolicyRepository_ByPolicyId;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.cloudfoundry.autoscaler.exceptions.PolicyNotFoundException;
import org.ektorp.CouchDbConnector;

public class ScalingPolicyRepositoryCollection extends
		TypedRepoCollection<AutoScalerPolicy> {

	private static final Logger logger = Logger
			.getLogger(ScalingPolicyRepositoryCollection.class);

	private ScalingPolicyRepository_All scalingPolicyRepo_all;
	private ScalingPolicyRepository_ByPolicyId scalingPolicyRepo_byPolicyId;

	public ScalingPolicyRepositoryCollection(CouchDbConnector db) {
		scalingPolicyRepo_all = new ScalingPolicyRepository_All(db);
		scalingPolicyRepo_byPolicyId = new ScalingPolicyRepository_ByPolicyId(
				db);

	}

	public ScalingPolicyRepositoryCollection(CouchDbConnector db,
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

		repoList.add(scalingPolicyRepo_all);
		repoList.add(scalingPolicyRepo_byPolicyId);
		return repoList;

	}

	@Override
	public TypedCouchDbRepositorySupport getDefaultRepo() {
		return scalingPolicyRepo_all;
	}

	public AutoScalerPolicy findByPolicyId(String policyId)
			throws PolicyNotFoundException {
		try {
			return scalingPolicyRepo_byPolicyId.findByPolicyId(policyId);
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

	public List<AutoScalerPolicy> getAllRecords() {
		try {
			return this.scalingPolicyRepo_all.getAllRecords();
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
		}
		return null;
	}

}
