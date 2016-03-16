package org.cloudfoundry.autoscaler.manager;

import java.util.List;

import org.cloudfoundry.autoscaler.data.couchdb.document.AutoScalerPolicy;
import org.cloudfoundry.autoscaler.exceptions.DataStoreException;
import org.cloudfoundry.autoscaler.exceptions.PolicyNotFoundException;

public interface PolicyManager {
	
	/**
	 * Creates a policy
	 * @param policy
	 * @return policyId
	 * @throws DataStoreException
	 */
	public String createPolicy(AutoScalerPolicy policy) throws DataStoreException;
	
	/**
	 * Updates a policy
	 * @param policy
	 * @throws DataStoreException
	 */
	public void updatePolicy(AutoScalerPolicy policy) throws DataStoreException;
	
	/**
	 * Gets scaling policy by id
	 * @param policyId
	 * @return
	 * @throws PolicyNotFoundException
	 * @throws DataStoreException
	 */
	public AutoScalerPolicy getPolicyById(String policyId) throws PolicyNotFoundException, DataStoreException;
	
	/**
	 * Deletes a policy
	 * @param policyId
	 * @throws DataStoreException
	 * @throws PolicyNotFoundException
	 */
	public void deletePolicy(String policyId) throws DataStoreException, PolicyNotFoundException;
	

	public void recoverMonitoredCache() throws Exception;
	
	public List<AutoScalerPolicy> getMonitoredCache();
	
	public void invalidateCache();
	
}
