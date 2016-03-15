package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.couchdb;

import java.util.ArrayList;
import java.util.List;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao.AutoScalerPolicyDAO;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.AutoScalerPolicy;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.repository.template.TypedCouchDbRepositorySupport;
import org.cloudfoundry.autoscaler.exceptions.PolicyNotFoundException;
import org.ektorp.CouchDbConnector;
import org.ektorp.support.View;

public class AutoScalerPolicyDAOImpl extends CommonDAOImpl implements AutoScalerPolicyDAO {
	@View(name = "byAll", map = "function(doc) { if (doc.type == 'AutoScalerPolicy' ) emit( [doc.policyId, doc.instanceMinCount, doc.instanceMaxCount], doc._id )}")
	public class ScalingPolicyRepository_All extends TypedCouchDbRepositorySupport<AutoScalerPolicy> {

		public ScalingPolicyRepository_All(CouchDbConnector db) {
			super(AutoScalerPolicy.class, db, "AutoScalerPolicy_byAll");
		}

		public List<AutoScalerPolicy> getAllRecords() {
			return queryView("byAll");
		}

	}

	@View(name = "by_policyId", map = "function(doc) { if(doc.type=='AutoScalerPolicy' && doc.policyId) {emit(doc.policyId, doc._id)}}")
	public class ScalingPolicyRepository_ByPolicyId extends TypedCouchDbRepositorySupport<AutoScalerPolicy> {

		public ScalingPolicyRepository_ByPolicyId(CouchDbConnector db) {
			super(AutoScalerPolicy.class, db, "AutoScalerPolicy_ByPolicyId");
		}

		public AutoScalerPolicy findByPolicyId(String policyId) throws PolicyNotFoundException {
			if (policyId == null)
				return null;

			List<AutoScalerPolicy> policies = queryView("by_policyId", policyId);
			if (policies == null || policies.size() == 0) {
				throw new PolicyNotFoundException(policyId);
			}
			return policies.get(0);
		}

	}

	private static final Logger logger = Logger.getLogger(AutoScalerPolicyDAOImpl.class);
	private ScalingPolicyRepository_All policyRepoAll = null;
	private ScalingPolicyRepository_ByPolicyId policyRepoByPolicyId = null;

	public AutoScalerPolicyDAOImpl(CouchDbConnector db) {

		this.policyRepoAll = new ScalingPolicyRepository_All(db);
		this.policyRepoByPolicyId = new ScalingPolicyRepository_ByPolicyId(db);

	}

	public AutoScalerPolicyDAOImpl(CouchDbConnector db, boolean initDesignDocument) {
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
	public List<AutoScalerPolicy> findAll() {
		// TODO Auto-generated method stub
		return this.policyRepoAll.getAllRecords();
	}

	@Override
	public AutoScalerPolicy findByPolicyId(String policyId) throws PolicyNotFoundException {
		// TODO Auto-generated method stub
		return this.policyRepoByPolicyId.findByPolicyId(policyId);
	}

	@Override
	public List<AutoScalerPolicy> getAllRecords() {
		return this.policyRepoAll.getAllRecords();
	}

	@SuppressWarnings("unchecked")
	@Override
	public <T> TypedCouchDbRepositorySupport<T> getDefaultRepo() {
		// TODO Auto-generated method stub
		return (TypedCouchDbRepositorySupport<T>) this.policyRepoAll;
	}

	@SuppressWarnings("unchecked")
	@Override
	public <T> List<TypedCouchDbRepositorySupport<T>> getAllRepos() {
		// TODO Auto-generated method stub
		List<TypedCouchDbRepositorySupport<T>> repoList = new ArrayList<TypedCouchDbRepositorySupport<T>>();
		repoList.add((TypedCouchDbRepositorySupport<T>) this.policyRepoAll);
		repoList.add((TypedCouchDbRepositorySupport<T>) this.policyRepoByPolicyId);
		return repoList;
	}

}
