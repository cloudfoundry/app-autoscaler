

package org.cloudfoundry.autoscaler.manager;

import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.common.util.CloudFoundryManager;
import org.cloudfoundry.autoscaler.constant.Constants;
import org.cloudfoundry.autoscaler.data.AutoScalingDataStore;
import org.cloudfoundry.autoscaler.data.couchdb.AutoScalingDataStoreFactory;
import org.cloudfoundry.autoscaler.data.couchdb.document.Application;
import org.cloudfoundry.autoscaler.data.couchdb.document.AutoScalerPolicy;
import org.cloudfoundry.autoscaler.exceptions.AppNotFoundException;
import org.cloudfoundry.autoscaler.exceptions.CloudException;
import org.cloudfoundry.autoscaler.exceptions.DataStoreException;
import org.cloudfoundry.autoscaler.exceptions.MetricNotSupportedException;
import org.cloudfoundry.autoscaler.exceptions.MonitorServiceException;
import org.cloudfoundry.autoscaler.exceptions.NoAttachedPolicyException;
import org.cloudfoundry.autoscaler.exceptions.PolicyNotFoundException;
import org.cloudfoundry.autoscaler.exceptions.TriggerNotSubscribedException;
/**
 * Implements the interface ApplicationManager
 * 
 *
 */
public class ApplicationManagerImpl implements ApplicationManager {
	private static final String CLASS_NAME = ApplicationManagerImpl.class.getName();
	private static final Logger logger     = Logger.getLogger(CLASS_NAME); 
	private static final ApplicationManagerImpl instance = new ApplicationManagerImpl();
	
	private ConcurrentHashMap<String, Application> applicationCache = new ConcurrentHashMap<String, Application>();
	private ApplicationManagerImpl(){
		
	}
	public static ApplicationManagerImpl getInstance(){
		return instance;
	}
	@Override
	public void addApplication(NewAppRequestEntity newAppData)
			throws PolicyNotFoundException, MetricNotSupportedException,
			DataStoreException, MonitorServiceException, CloudException{
		logger.info("Add application");
		String appId = newAppData.getAppId();
		String orgId = newAppData.getOrgId();
		String spaceId = newAppData.getSpaceId();

		if (orgId == null || spaceId == null){
			logger.info("Call CF Rest API to get org and space of application " + appId);
			//Gets org space
			Map<String, String> orgSpace;
			try {
				orgSpace = CloudFoundryManager.getInstance().getOrgSpaceByAppId(appId);
				orgId = orgSpace.get("ORG_GUID");
				spaceId = orgSpace.get("SPACE_GUID");			
			} catch (Exception e) {
				logger.error ("Fail to add application since no valid org/space info for app " + appId);
				return;
			}
		}

		Application app = new Application(appId, newAppData.getServiceId(),
				newAppData.getBindingId(), orgId, spaceId);
		app.setState(Constants.APPLICATION_STATE_ENABLED);
		app.setBindTime(System.currentTimeMillis());
		AutoScalingDataStore dataStore = AutoScalingDataStoreFactory
				.getAutoScalingDataStore();
		Application oldApp = dataStore.getApplication(appId);
		if (oldApp != null && oldApp.getPolicyId() != null){
			app.setPolicyId(oldApp.getPolicyId());
			app.setPolicyState(oldApp.getPolicyState());
		}
		String policyId = app.getPolicyId();
		if (policyId != null && isPolicyEnabled(app)) {
			// get policy
			PolicyManager policyManager = PolicyManagerImpl.getInstance();
			AutoScalerPolicy policy = null;
			policy = policyManager.getPolicyById(policyId);
			try {
				handleInstancesByPolicy(appId, policy); // Check the maximum instances and minimum
														// instances
			} catch (AppNotFoundException e) {
				logger.warn("The application " + appId
						+ " doesn't finish staging yet. ", e);
			} catch (Exception e) {
				logger.warn(
						"Error happens when handle instance number by policy. ",
						e);
			}
			TriggerManager tr = new TriggerManager(appId, policy);
			// subscribe trigger
			tr.subscribeTriggers();
		}
		// Store application
		dataStore.saveApplication(app);
		applicationCache.put(appId, app);
		
	}

	@Override
	public void removeApplicationByBindingId(String bindingId) throws DataStoreException, PolicyNotFoundException, MonitorServiceException, NoAttachedPolicyException {
		AutoScalingDataStore dataStore = AutoScalingDataStoreFactory
				.getAutoScalingDataStore();
		Application app = dataStore.getApplicationByBindingId(bindingId);

		if (app!=null){
		try {

				TriggerManager tr = new TriggerManager(app.getAppId(), null);
				tr.unSubscribeTriggers();
			
			} catch (TriggerNotSubscribedException e) {
				logger.warn("Trigger not found on monitor service.");
			} 
			app.setState(Constants.APPLICATION_STATE_UNBOUND);
			dataStore.saveApplication(app);
			applicationCache.put(app.getAppId(), app);
			
		}
		else {
			logger.error ("ERRROR: Can't find expected App with binding id " + bindingId) ;
		}
	}

	@Override
	public Application getApplicationByBindingId(String bindingId) throws DataStoreException {
		AutoScalingDataStore dataStore = AutoScalingDataStoreFactory
				.getAutoScalingDataStore();
		return dataStore.getApplicationByBindingId(bindingId);
	}

	@Override
	public void updatePolicyOfApplication(String appId, String policyState, AutoScalerPolicy policy) throws MonitorServiceException, MetricNotSupportedException {
		try {
			if (AutoScalerPolicy.STATE_ENABLED.equals(policyState)){
				logger.info("The policy is enabled for application " + appId);
				handleInstancesByPolicy(appId, policy); //Check the maximum instances and minimum instances
			}
			else{
				logger.info("The policy is disabled for application " + appId);
			}
			
		} catch (AppNotFoundException e) {
			logger.warn( "The application " + appId + " doesn't finish staging yet. ", e);
		} catch (DataStoreException e) {
			logger.warn( "The data store can not be accessed. ", e);
		} catch (Exception e) {
			logger.warn( "Error happens when handle handle instance number by policy. ", e);
		}
		TriggerManager tr = new TriggerManager(appId, policy);
		try {
			tr.unSubscribeTriggers();
		} catch (TriggerNotSubscribedException e) {
			//Ignore
		}
		if (AutoScalerPolicy.STATE_ENABLED.equals(policyState))
			tr.subscribeTriggers();
		
	}

	@Override
	public List<Application> getApplications(String serviceId) throws DataStoreException {
		AutoScalingDataStore dataStore = AutoScalingDataStoreFactory
				.getAutoScalingDataStore();
		return dataStore.getApplications(serviceId);
	}
	public void handleInstancesByPolicy(String appId, AutoScalerPolicy policy) throws Exception{
		ApplicationScaleManager manager = ApplicationScaleManagerImpl.getInstance();
		manager.doScaleBySchedule(appId, policy);
	}
	@Override
	public Application getApplication(String appId) throws DataStoreException, CloudException {
		
		Application app = null;
		if (applicationCache.containsKey(appId)){ 
			app = applicationCache.get(appId);
		} else {
			app = AutoScalingDataStoreFactory.getAutoScalingDataStore().getApplication(appId);
			applicationCache.put(appId, app);
		}
		
		if (app!= null && app.getAppType() == null){
			try {
				String appType = CloudFoundryManager.getInstance().getAppType(appId);
				app.setAppType(appType);
				AutoScalingDataStoreFactory.getAutoScalingDataStore().saveApplication(app);
				applicationCache.put(appId, app);
			} catch (Exception e) {
				logger.error("Failed to get the app type for app " + appId, e);
			}
				
		}
		return app;
	}
	@Override
	public void attachPolicy(String appId, String policyId, String policyState) throws DataStoreException, MonitorServiceException, MetricNotSupportedException, PolicyNotFoundException {
		AutoScalingDataStore dataStore = AutoScalingDataStoreFactory
				.getAutoScalingDataStore();
		Application app = dataStore.getApplication(appId);
		app.setPolicyId(policyId);
		app.setPolicyState(policyState);
		dataStore.saveApplication(app);	
		applicationCache.put(appId,app);
		PolicyManager policyManager = PolicyManagerImpl.getInstance();
		AutoScalerPolicy policy = policyManager.getPolicyById(policyId);
		updatePolicyOfApplication(appId, policyState, policy);
	}
	@Override
	public void detachPolicy(String appId, String policyId, String policyState) throws DataStoreException, MonitorServiceException, MetricNotSupportedException, PolicyNotFoundException {
			AutoScalingDataStore dataStore = AutoScalingDataStoreFactory
					.getAutoScalingDataStore();
			Application app = dataStore.getApplication(appId);
			app.setPolicyId(null);
			app.setPolicyState(null);
			dataStore.saveApplication(app);	
			applicationCache.put(appId,app);
			PolicyManager policyManager = PolicyManagerImpl.getInstance();
			AutoScalerPolicy policy = policyManager.getPolicyById(policyId);
			TriggerManager tr = new TriggerManager(appId, policy);
			try {
				tr.unSubscribeTriggers();
			} catch (TriggerNotSubscribedException e) {
				//Ignore
			}
		}
	@Override
	public List<Application> getApplicationByPolicyId(String policyId)
			throws DataStoreException {
		AutoScalingDataStore dataStore = AutoScalingDataStoreFactory
				.getAutoScalingDataStore();
		return dataStore.getApplicationsByPolicyId(policyId);
	}


	
	/**
	 * Checks if app is enabled
	 * @param app
	 * @return
	 */
	private boolean isPolicyEnabled(Application app){
		if (AutoScalerPolicy.STATE_ENABLED.equals(app.getPolicyState())){
			return true;
		}else
			return false;
	}
	
	public void invalidateCache(){
		applicationCache.clear();
	}
}