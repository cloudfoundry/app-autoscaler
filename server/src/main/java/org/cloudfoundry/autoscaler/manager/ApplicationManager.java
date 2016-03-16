

package org.cloudfoundry.autoscaler.manager;

import java.util.List;

import org.cloudfoundry.autoscaler.data.couchdb.document.Application;
import org.cloudfoundry.autoscaler.data.couchdb.document.AutoScalerPolicy;
import org.cloudfoundry.autoscaler.exceptions.AppNotFoundException;
import org.cloudfoundry.autoscaler.exceptions.CloudException;
import org.cloudfoundry.autoscaler.exceptions.DataStoreException;
import org.cloudfoundry.autoscaler.exceptions.MetricNotSupportedException;
import org.cloudfoundry.autoscaler.exceptions.MonitorServiceException;
import org.cloudfoundry.autoscaler.exceptions.NoAttachedPolicyException;
import org.cloudfoundry.autoscaler.exceptions.NoMonitorServiceBoundException;
import org.cloudfoundry.autoscaler.exceptions.PolicyNotFoundException;
import org.cloudfoundry.autoscaler.exceptions.TriggerNotSubscribedException;

/**
 * Manage applications
 * 
 * 
 * 
 */
public interface ApplicationManager {
	/**
	 * Adds an application to scaling service
	 * 
	 * @param newAppData
	 * @throws MonitorServiceException 
	 * @throws NoMonitorServiceBoundException 
	 * @throws CloudException 
	 * @throws AppNotFoundException 
	 */
	public void addApplication(NewAppRequestEntity newAppData)
			throws PolicyNotFoundException, MetricNotSupportedException,
			DataStoreException, MonitorServiceException, NoMonitorServiceBoundException, CloudException, AppNotFoundException;
	/**
	 * Attach policy to an application
	 * @param appId
	 * @param policy
	 * @param state
	 * @throws DataStoreException 
	 * @throws NoMonitorServiceBoundException 
	 * @throws MetricNotSupportedException 
	 * @throws MonitorServiceException 
	 * @throws PolicyNotFoundException 
	 */
	public void attachPolicy (String appId, String policy, String state) throws DataStoreException, MonitorServiceException, MetricNotSupportedException, PolicyNotFoundException;

	/**
	 * Detach policy from an application
	 * @param appId
	 * @param policy
	 * @param state
	 * @throws DataStoreException 
	 * @throws NoMonitorServiceBoundException 
	 * @throws MetricNotSupportedException 
	 * @throws MonitorServiceException 
	 * @throws PolicyNotFoundException 
	 */
	public void detachPolicy (String appId, String policy, String state) throws DataStoreException, MonitorServiceException, MetricNotSupportedException, PolicyNotFoundException;

	
	/**
	 * Removes an application from scaling service
	 * @throws NoMonitorServiceBoundException 
	 * @throws TriggerNotSubscribedException 
	 */
	public void removeApplicationByBindingId(String bindingId)  throws DataStoreException, PolicyNotFoundException , 
	MonitorServiceException, NoAttachedPolicyException;
	
	/**
	 * Gets application id by bindingId
	 * @param bindingId
	 * @return
	 */
	public Application getApplicationByBindingId(String bindingId)  throws DataStoreException;
	
	/**
	 * Updates policy of an application
	 * @param appId
	 * @param policy
	 * @throws MonitorServiceException 
	 * @throws MetricNotSupportedException 
	 * @throws NoMonitorServiceBoundException 
	 */
	public void updatePolicyOfApplication(String appId, String policyState, AutoScalerPolicy policy) throws MonitorServiceException, MetricNotSupportedException, NoMonitorServiceBoundException;
	
	
	/**
	 * Gets the applications that are bound to this service
	 * @param serviceId
	 * @return
	 * @throws DataStoreException 
	 */
	public List<Application> getApplications(String serviceId) throws DataStoreException;
	
	/**
	 * Gets cloud application
	 * @param appId
	 * @return
	 * @throws CloudException 
	 * @throws AppNotFoundException 
	 * @throws DataStoreException 
	 */
	public Application getApplication(String appId) throws DataStoreException, CloudException;

	/**
	 * Gets applications by policy id
	 * @param policyId
	 * @return
	 */
	public List<Application> getApplicationByPolicyId(String policyId)  throws DataStoreException;
	
	public void handleInstancesByPolicy(String appId, AutoScalerPolicy policy) throws  Exception;
	
	public void invalidateCache();
}
