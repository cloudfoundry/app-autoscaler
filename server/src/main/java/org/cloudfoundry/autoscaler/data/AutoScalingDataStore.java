package org.cloudfoundry.autoscaler.data;

import java.util.List;
import java.util.Map;
import java.util.Set;

import org.cloudfoundry.autoscaler.bean.Trigger;
import org.cloudfoundry.autoscaler.data.couchdb.document.AppAutoScaleState;
import org.cloudfoundry.autoscaler.data.couchdb.document.AppInstanceMetrics;
import org.cloudfoundry.autoscaler.data.couchdb.document.Application;
import org.cloudfoundry.autoscaler.data.couchdb.document.AutoScalerPolicy;
import org.cloudfoundry.autoscaler.data.couchdb.document.BoundApp;
import org.cloudfoundry.autoscaler.data.couchdb.document.ScalingHistory;
import org.cloudfoundry.autoscaler.data.couchdb.document.ServiceConfig;
import org.cloudfoundry.autoscaler.data.couchdb.document.TriggerRecord;
import org.cloudfoundry.autoscaler.exceptions.DataStoreException;
import org.cloudfoundry.autoscaler.exceptions.PolicyNotFoundException;
import org.cloudfoundry.autoscaler.manager.ScalingHistoryFilter;

public interface AutoScalingDataStore {
	
	/**
	 * Saves application
	 * @param app
	 */
	public void saveApplication(Application app) throws DataStoreException;
	
	/**
	 * Remove application
	 * @param appId
	 */
	public void removeApplication(String appId);
	
	/**
	 * Gets application by bindingId
	 * @param bindingId
	 */
	public Application getApplicationByBindingId(String bindingId);

	/**
	 * Gets application
	 * @param appId
	 * @return
	 */
	public Application getApplication(String appId);
	/**
	 * Removes application by bindingId
	 * @param bindingId
	 * @return
	 */
	public void removeApplicationByBindingId(String bindingId)  throws DataStoreException;
	
	/**
	 * Get the policy of an app
	 * @param policyId
	 * @return
	 */
	public AutoScalerPolicy getPolicyById(String policyId) throws PolicyNotFoundException;
	
	/**
	 * Save policy
	 * @param policy
	 */
	public String savePolicy(AutoScalerPolicy policy)  throws DataStoreException;
	
	/**
	 * Deletes a policy
	 * @param policyId
	 * @throws DataStoreException
	 */
	public void deletePolicy(String policyId) throws DataStoreException ,PolicyNotFoundException;
	
	/**
	 * Save scaling state
	 */
	public void saveScalingState(AppAutoScaleState state)  throws DataStoreException ;
	
	/**
	 * Get scaling state
	 * @param appId
	 */
	public AppAutoScaleState getScalingState(String appId);
	
	/**
	 * Gets applications that are bound to this service
	 * @param serviceId
	 * @return
	 */
	public List<Application> getApplications(String serviceId);
	
	/**
	 * Gets applications by policy id
	 * @param policyId
	 * @return
	 */
	public List<Application> getApplicationsByPolicyId(String policyId);
	
	
	/**
	 * Save scaling history
	 * @param scalingHistory
	 * @throws DataStoreException 
	 */
	public void saveScalingHistory(ScalingHistory scalingHistory) throws DataStoreException;
	
	/**
	 * Get scaling history list
	 * @param filter
	 * @return
	 * @throws DataStoreException 
	 */
	public List<ScalingHistory> getHistoryList(ScalingHistoryFilter filter) throws DataStoreException;
	
	/**
	 * Get scaling history list
	 * @param filter
	 * @return
	 * @throws DataStoreException 
	 */
	public int getHistoryCount(ScalingHistoryFilter filter) throws DataStoreException;
	
	/**
	 * Gets history by id
	 * @param id
	 * @return
	 * @throws DataStoreException
	 */
	public ScalingHistory getHistoryById(String id) throws DataStoreException;

	
	
	
	
	
	/**
	 * Get AutoScalerPolicy list
	 * @return
	 */
	public List<AutoScalerPolicy> getAutoScalerPolicies();
	
    public void addTrigger(Trigger t) throws Exception;

    public void removeTrigger(String appId) throws Exception;

    public Map<String, List<TriggerRecord>> getAllTriggers() throws Exception;

    public void addAppStats(AppInstanceMetrics appInstanceMetrics) throws Exception;

    public void removeAppStatsWithHistory(String appId) throws Exception;

    public  List<AppInstanceMetrics> getAppStatsHistoryByAppIdAfter(String appId, long newerThan/*, Set<Integer> runningInstIndexes*/)
            throws Exception;

    public  ServiceConfig getConfig(String serviceId) throws Exception;



    public  long getSmallestPersistTime();

    public  List<ServiceConfig> getAllServiceConfigs() throws Exception;

    public  void updateBinding(String serviceId, String appId, String appType, String appName) throws Exception;

    public  void removeBinding(String serviceId, String appId) throws Exception;

    public  Map<String, List<BoundApp>> getAllBindings() throws Exception;

    public  Set<String> getAllBoundServiceIds() throws Exception;

    public  List<BoundApp> getAllBindingsByServiceId(String serviceId) throws Exception;

    public  String getAppTypeById(String appId) throws Exception;

//    public  long getReportInterval(String serviceId) throws Exception;

    public  void addBinding(String serviceId, String appId, String appType, String appName) throws Exception;

	
}
