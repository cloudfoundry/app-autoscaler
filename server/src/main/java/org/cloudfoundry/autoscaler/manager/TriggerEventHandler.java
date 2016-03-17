package org.cloudfoundry.autoscaler.manager;

import java.util.List;
import java.util.UUID;

import org.apache.log4j.Level;
import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.Constants;
import org.cloudfoundry.autoscaler.ScalingStateMonitor;
import org.cloudfoundry.autoscaler.ScalingStateMonitorTask;
import org.cloudfoundry.autoscaler.bean.AutoScalerPolicyTrigger;
import org.cloudfoundry.autoscaler.bean.MonitorTriggerEvent;
import org.cloudfoundry.autoscaler.data.AutoScalingDataStore;
import org.cloudfoundry.autoscaler.data.couchdb.AutoScalingDataStoreFactory;
import org.cloudfoundry.autoscaler.data.couchdb.document.AppAutoScaleState;
import org.cloudfoundry.autoscaler.data.couchdb.document.Application;
import org.cloudfoundry.autoscaler.data.couchdb.document.AutoScalerPolicy;
import org.cloudfoundry.autoscaler.exceptions.AppNotFoundException;
import org.cloudfoundry.autoscaler.exceptions.CloudException;
import org.cloudfoundry.autoscaler.exceptions.DataStoreException;
import org.cloudfoundry.autoscaler.exceptions.NoAttachedPolicyException;
import org.cloudfoundry.autoscaler.exceptions.PolicyNotFoundException;
import org.cloudfoundry.autoscaler.metric.util.CloudFoundryManager;
import org.cloudfoundry.autoscaler.metric.util.ConfigManager;
import org.cloudfoundry.autoscaler.util.IcapMonitorMetricsMapper;

/**
 * Handles trigger events
 * 
 * 
 * 
 */
public class TriggerEventHandler {
	private static final Logger logger = Logger
			.getLogger(TriggerEventHandler.class.getName());
	private static final Logger loggerEvent = Logger.getLogger("triggerevent");
	
	public final static int TRIGGER_TYPE_MONITOR_EVENT = 0;
	public final static int TRIGGER_TYPE_POLICY_CHANGED = 1;
	public final long eventTimeout = Long.parseLong(ConfigManager.get("LAST_TRIGGER_EVENT_TIME_OUT", "10")) * 60 * 1000L;
	private AutoScalingDataStore stateStore;
	// we use constants instead of an enum because the app-state variable
	// instanceCountState is stored
	// in a state store (a DB), which may lose the enum context when reading it
	// back in; just simpler this way
	private MonitorTriggerEvent triggerEvent;
	private String appId;
	private AppAutoScaleState appState;
	private String triggerId;
	private AutoScalerPolicyTrigger policyTrigger;
	private AutoScalerPolicy policy;
	private int currentInstances;
	private ScalingStateManager stateManager;
	private long busyTimestamp;
	private long approximateCooldownSetting;
	private long approximateLastScalingActionTimeStamps;
	private boolean busy;


	public TriggerEventHandler() {
		this.approximateCooldownSetting = 0l;
		this.approximateLastScalingActionTimeStamps = 0l;
		this.busyTimestamp = 0l;
	}

	public TriggerEventHandler(MonitorTriggerEvent event)
			throws AppNotFoundException, PolicyNotFoundException,
			CloudException, DataStoreException {
		initialize(event);
	}

	private void initialize(MonitorTriggerEvent event)   {
		stateStore = AutoScalingDataStoreFactory.getAutoScalingDataStore();
		stateManager = ScalingStateManager.getInstance();
		triggerEvent = event;
		appId = event.getAppId();
		triggerId = event.getTriggerId();
		appState = null;
		policy = null;
		policyTrigger = null;
	}

	public void handleEvent(MonitorTriggerEvent event) {

		if ( approximateLastScalingActionTimeStamps + approximateCooldownSetting > System.currentTimeMillis()) {
			loggerEvent.debug("Event " + event.toString() + " for " + this.appId + " is ignored as it happens in cooldown period. ");
			return;
		}
		
		if (isBusy()){
			loggerEvent.debug("Event " + event.toString() + " for " + this.appId + " is ignored as the related handler is busy.");
			return;
		}
		setBusy(true);
		initialize(event);

		if (loggerEvent.getLevel() == Level.DEBUG){
			long before=System.currentTimeMillis();
			handleEvent();
			long duration=System.currentTimeMillis() - before;
			loggerEvent.debug("Duration " + duration + " to handle event " + event.toString() + " for appId " + appId);
		} else {
			handleEvent();
		}
		
		setBusy(false);
	}

	public synchronized boolean isBusy(){
		//if a httpconnection is hang (i.e. in jersey client) when handle event, the eventTimeout definition will help to unlock the thread.
		if ((busy) && (System.currentTimeMillis() - this.busyTimestamp < this.eventTimeout))
			return true;
		else 
			return false;
	}
	
	public synchronized void setBusy(boolean busy){
		if (busy){
			this.busy = true;
			this.busyTimestamp = System.currentTimeMillis();
		}
		else {
			this.busy = false;
			this.busyTimestamp = 0;
		}	
	}
	
	/**
	 * Handles event
	 * 
	 * @throws AppNotFoundException
	 * @throws CloudException
	 */
	private void handleEvent()  {

			logger.debug("Receive trigger " + triggerEvent.toString() + "with handler " + this.toString() + " for appId = " + appId 	+ ", triggerId = " + triggerId + ", metricType = " + triggerEvent.getMetricType());
			loggerEvent.debug("Receive trigger " + triggerEvent.toString() + "with handler " + this.toString() + " for appId = " + appId 	+ ", triggerId = " + triggerId + ", metricType = " + triggerEvent.getMetricType());

		 	try {
				try {
					policy = getPolicy(triggerEvent.getAppId());
					policyTrigger = getAutoScalerPolicyTrigger(policy, triggerEvent);
					if (policyTrigger == null) {
						logger.warn("PolicyTrigger is null.");
						return;
					}

					approximateCooldownSetting = 1000L * ((policyTrigger.getStepDownCoolDownSecs() < policyTrigger.getStepUpCoolDownSecs() ) ?
							policyTrigger.getStepDownCoolDownSecs() : policyTrigger.getStepUpCoolDownSecs());
					
				} catch (PolicyNotFoundException e) {
					logger.error("The policy for app " + appId + " .can not be found.", e);
					return;
				} catch (DataStoreException e) {
					logger.error("Error occurs when getting the policy for app: " + appId, e);
					return;
				}  catch (CloudException e) {
					logger.error("Failed to get find the application ", e);
				} catch (Exception e) {
					logger.error("The service doesn't have an attached policy.", e);
					return;
				}

				if(! validateCooldownSetting()) {
					logger.debug("Abort trigger " + triggerEvent.toString() + " for event " + triggerEvent.getMetricType() + "/" + triggerId + " as a scaling action is ongoing or cooldown time " + appId);
					return;
				}
				
				int newCount = 0;
				if (validateInstanceCounts()){
					newCount = calculateNewCount(policyTrigger);
					logger.info("Handle monitor trigger" + this.toString() + " : appId = " + appId 	+ ", triggerId = " + triggerId + ", metricValue = " + triggerEvent.getMetricValue());
					logger.info("Scale: Target instance count for app " + appId + " is " + newCount);
					String actionUUID = UUID.randomUUID().toString();
					stateManager.setScalingStateRealizing(appId,
								triggerId, currentInstances, newCount,
								policyTrigger, TRIGGER_TYPE_MONITOR_EVENT,
								actionUUID , null, policy.getTimezone(), null, null);
					if (startScaling(newCount, actionUUID))
						approximateLastScalingActionTimeStamps = System.currentTimeMillis();
				} else {
					logger.debug("Abort trigger " + triggerEvent.toString() + " for event " + triggerEvent.getMetricType() + "/" + triggerId + " as it reachs the max/min instance count. " + appId);
					return; 
				}
				
			} catch (Exception e) {
				logger.error("Error occurs when handle trigger event for  app: " + appId, e);
				return;
			}

	}

	/**
	 * Checks if min instance count < app currnent instance count < max instance count
	 * 
	 * @return true if valid to scale in/out
	 * @throws Exception 
	 */

	private boolean validateInstanceCounts() throws Exception{
		try {
			currentInstances =  CloudFoundryManager.getInstance().getAppInstancesByAppId(appId);
		} catch (AppNotFoundException e) {
			throw new AppNotFoundException("Application " + appId
					+ " can not be found.");
		} catch (CloudException e) {
			logger.error("Failed to get application instances", e);
			return false;
		} 
		
		/** Check minimum count and maximum count **/
		if (AutoScalerPolicyTrigger.TriggerId_LowerThreshold.equals(triggerId)
				&& currentInstances <= policy.getCurrentInstanceMinCount()) {
			logger.debug("False: Min count reached. No scaling in action.");
			return false;
		} else if (AutoScalerPolicyTrigger.TriggerId_UpperThreshold
				.equals(triggerId)
				&& currentInstances >= policy.getCurrentInstanceMaxCount()) {
			logger.debug("False: Max count reached. No scaling out action.");
			return false;
		} 
		
		return true;
		
	}
	
	/**
	 * Checks if the app should be scaled in/out according to cooldown settings
	 * 
	 * @return true if should scale in/out
	 */
	private boolean validateCooldownSetting() {

		appState = stateStore.getScalingState(appId);
        
		if (appState == null) {
			// if this is the first trigger for this app we don't
			// have an app-state, so we create the initial one
			return true;
		} else if ( (appState.getInstanceCountState() != ScalingStateManager.SCALING_STATE_COMPLETED)
				&& (appState.getInstanceCountState() != ScalingStateManager.SCALING_STATE_FAILED)){
			long lastStartTime = appState.getLastActionStartTime();
	        long currentTime = System.currentTimeMillis();
	        boolean timeExpired = (currentTime - lastStartTime) > this.eventTimeout;
	        if(timeExpired){
	        	logger.debug("True: Last scaling action is not completed but it's time expired for application " + appId + ".");
	        	return true;
	        }
			logger.debug("False: Last scaling action is not completed for application " + appId + ".");
			return false;
		} else { 
			/** Check cool down time **/
			long cooldownEndtime = appState.getLastActionEndTime() + 1000L * getCooldownSecs();
			if (System.currentTimeMillis() < cooldownEndtime) {// in cooldown time
				logger.debug("False: It's cooldown time for application " + appId
					+ ". No scaling in action.");
				return false;
			}
		}
		return true;
	}

	private AutoScalerPolicyTrigger getAutoScalerPolicyTrigger(AutoScalerPolicy policy, MonitorTriggerEvent event)
			throws AppNotFoundException, PolicyNotFoundException {
		List<AutoScalerPolicyTrigger> triggers = policy.getPolicyTriggers();
		if (triggers == null)
			throw new PolicyNotFoundException("No policy for metric: "
					+ event.getMetricType());
		for (AutoScalerPolicyTrigger trigger : triggers) {
			String metric = IcapMonitorMetricsMapper.getMetricNameMapper().get(
					trigger.getMetricType().toUpperCase());
			if (metric.equalsIgnoreCase(event.getMetricType())) {
				return trigger;
			}
		}
		throw new PolicyNotFoundException("No policy for metric: "
				+ event.getMetricType());
	}

	private boolean startScaling(int newCount, String actionId)
			throws AppNotFoundException {

		logger.info("Start scaling for the application " + appId);
		// get instance step depending on trigger ID
		// tell the Cloud manager to set the new instance count
		try {
			CloudFoundryManager.getInstance().updateInstances(appId, newCount);
			ScalingStateMonitorTask task = new ScalingStateMonitorTask(appId,
					newCount, actionId);
			ScalingStateMonitor.getInstance().monitor(task);
			return true; //successfully to trigger scaling
		} catch (CloudException e2) {
			String errorCode = e2.getErrorCode();
			if (Constants.MemoryQuotaExceeded.equals(errorCode)) {
				logger.error("Failed to scale application "
						+ appId + ". You have exceeded your organization's memory limit.");
			} else {
				errorCode = Constants.CloudFoundryInternalError;
				logger.error("Failed to scale application " + appId + "." + e2.getMessage());
			}
			stateManager.setScalingStateFailed(appId, triggerId,
					currentInstances, currentInstances, policyTrigger,
					TRIGGER_TYPE_MONITOR_EVENT, errorCode, actionId,null,policy.getTimezone(),null, null);
			return false; 
		} catch (Exception e) {
			logger.error("Failed to update application " + appId + " to scaling state." + e.getMessage(), e);
			stateManager.setScalingStateFailed(appId, triggerId,
					currentInstances, currentInstances, policyTrigger,
					TRIGGER_TYPE_MONITOR_EVENT, e.getMessage(), actionId,null, policy.getTimezone(), null, null);
			return false;
		}
	}

	/*****************************************************************************************************************
	 * 
	 * @throws PolicyNotFoundException
	 * @throws DataStoreException
	 * @throws NoAttachedPolicyException
	 * @throws CloudException
	 * @throws AppNotFoundException 
	 */
	public AutoScalerPolicy getPolicy(String appId) throws PolicyNotFoundException, DataStoreException, CloudException, AppNotFoundException{
		ApplicationManager am = ApplicationManagerImpl.getInstance();
		Application app = am.getApplication(appId);
		if (app == null) 
			throw new AppNotFoundException(appId);
		String policyId = app.getPolicyId();
		AutoScalerPolicy policy = PolicyManagerImpl.getInstance().getPolicyById(policyId);
		if (policy == null) {
			throw new PolicyNotFoundException(appId);
		}
		return policy;
	}

	private long getCooldownSecs() {
		if (AutoScalerPolicyTrigger.TriggerId_LowerThreshold.equals(triggerId)) {
			return policyTrigger.getStepDownCoolDownSecs();
		} else
			return policyTrigger.getStepUpCoolDownSecs();
	}

	private int calculateNewCount(AutoScalerPolicyTrigger policyTrigger) {
		int newCount = 0;
		int instanceStep = policyTrigger.getInstanceStepCountUp();
		String adjustmentType = null;
		if (AutoScalerPolicyTrigger.TriggerId_LowerThreshold.equals(triggerId)) {
			instanceStep = policyTrigger.getInstanceStepCountDown(); // is a negative value 
			adjustmentType = policyTrigger.getScaleInAdjustmentType();
		} else
			adjustmentType = policyTrigger.getScaleOutAdjustmentType();
		if (AutoScalerPolicyTrigger.ADJUSTMENT_CHANGE_PERCENTAGE
				.equals(adjustmentType)) {
			int adjustment = instanceStep * currentInstances / 100;
			if (adjustment == 0) {
				if (instanceStep < 0)
					adjustment = -1;
				else
					adjustment = 1;
			}
			newCount = currentInstances + adjustment;
		} else
			newCount = currentInstances + instanceStep;
		if (newCount < policy.getCurrentInstanceMinCount()) {
			newCount = policy.getCurrentInstanceMinCount();
		} else if (newCount > policy.getCurrentInstanceMaxCount()) {
			newCount = policy.getCurrentInstanceMaxCount();
		}
		return newCount;
	}
}
