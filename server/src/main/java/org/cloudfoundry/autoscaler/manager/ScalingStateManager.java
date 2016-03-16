package org.cloudfoundry.autoscaler.manager;


import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.Constants;
import org.cloudfoundry.autoscaler.bean.AutoScalerPolicyTrigger;
import org.cloudfoundry.autoscaler.data.AutoScalingDataStore;
import org.cloudfoundry.autoscaler.data.couchdb.AutoScalingDataStoreFactory;
import org.cloudfoundry.autoscaler.data.couchdb.document.AppAutoScaleState;
import org.cloudfoundry.autoscaler.data.couchdb.document.ScalingHistory;
import org.cloudfoundry.autoscaler.exceptions.DataStoreException;

public class ScalingStateManager {
	private static final String CLASS_NAME = ScalingStateManager.class
			.getName();
	private static final Logger logger = Logger.getLogger(CLASS_NAME);
	public static final int SCALING_STATE_READY = 1;
	public static final int SCALING_STATE_REALIZING = 2;
	public static final int SCALING_STATE_COMPLETED = 3;
	public static final int SCALING_STATE_FAILED = -1;

	private AutoScalingDataStore dataStore = null;
	private ScalingHistoryManager historyStore = null;

	private ScalingStateManager()  {
		dataStore = AutoScalingDataStoreFactory.getAutoScalingDataStore();
		historyStore= ScalingHistoryManager.getInstance();
	}

	public static ScalingStateManager getInstance() {
		return new ScalingStateManager();
	}
	
	/**
	 * Set scaling state to realizing state
	 * @param appId
	 * @param scalingState
	 * @param thresholdType
	 * @param currentInstance
	 * @param newCount
	 * @param policyTrigger
	 * @param triggerType
	 */
	public boolean setScalingStateRealizing(String appId, 
			String thresholdType, int currentInstance, int newCount, AutoScalerPolicyTrigger policyTrigger,int triggerType, String actionId, String scheduleType,String timeZone,  Long scheduleStartTime, Integer dayOfWeek) {
		

		// update app-state in store
		try {

			AppAutoScaleState appState = dataStore.getScalingState(appId);
			if (appState == null) {
				appState = new AppAutoScaleState(appId, SCALING_STATE_READY);
			}

			long startTime = System.currentTimeMillis();
			ScalingHistory history = buildScalingHistory(appId, SCALING_STATE_REALIZING, startTime, 0L, thresholdType,
					currentInstance, newCount, policyTrigger, triggerType, null, scheduleType, timeZone,
					scheduleStartTime, dayOfWeek);
			history.setId(actionId);

			// add scale event
			appState.setInstanceCountState(SCALING_STATE_REALIZING);
			appState.setLastActionTriggerId(thresholdType);
			appState.setLastActionInstanceTarget(newCount);
			appState.setLastActionStartTime(startTime);
			appState.setScaleEvent(history);
			dataStore.saveScalingState(appState);
			return true;

		} catch (DataStoreException e) {
			logger.error("Error occurs when store the state of application " + appId + ".", e);
			
		}
		try {
			Thread.sleep(10);
		} catch (InterruptedException e) {
		}

		return false;
	}

	/**
	 * Set scaling state to realizing state
	 * @param appId
	 * @param scalingState
	 * @param thresholdType
	 * @param currentInstance
	 * @param newCount
	 * @param policyTrigger
	 * @param triggerType
	 */
	public void setScalingStateFailed(String appId, 
			String thresholdType, int currentInstance, int newCount, AutoScalerPolicyTrigger policyTrigger,
			int triggerType, String errorCode, String actionId,String scheduleType,String timeZone, Long scheduleStartTime, Integer dayOfWeek) {
		
			try{
				AppAutoScaleState appState = dataStore.getScalingState(appId);
				if (appState == null) {
					appState = new AppAutoScaleState(appId,
							SCALING_STATE_READY);
				}
				long startTime = System.currentTimeMillis();
				long endTime = startTime + 5L;
				
				ScalingHistory lastScalingHistory = historyStore.getScalingHistory(appState.getHistoryId());
				if (lastScalingHistory != null
						&& lastScalingHistory.getStatus() == SCALING_STATE_FAILED
						&& errorCode.equals(lastScalingHistory.getErrorCode())
						&& lastScalingHistory.getInstances() == newCount) {
					ScalingHistory scaleState = appState.getScaleEvent();
					if(scaleState!=null && scaleState.getStartTime()!=0){
						endTime = startTime;
						startTime = scaleState.getStartTime();
					}
					
					ScalingHistory history = buildScalingHistory(appId,
							SCALING_STATE_FAILED, startTime, endTime, thresholdType, currentInstance, newCount,
							policyTrigger, triggerType, errorCode, scheduleType, timeZone, scheduleStartTime, dayOfWeek);
					
					history.setId(lastScalingHistory.getId());
					history.setRevision(lastScalingHistory.getRevision());
					historyStore.saveScalingHistory(history);
					appState.setInstanceCountState(SCALING_STATE_FAILED);
					appState.setLastActionTriggerId(thresholdType);
					appState.setLastActionInstanceTarget(newCount);
					appState.setLastActionStartTime(startTime);
					appState.setErrorCode(errorCode);
					appState.setScaleEvent(null);
					dataStore.saveScalingState(appState);
					return;
				}
				
				ScalingHistory history = historyStore.getScalingHistory(actionId);
				
				if (history == null) {
					history = appState.getScaleEvent();
				}
				
				if (history != null){
					if (!actionId.equals(history.getId())) {
						return;
					}
					history.setStatus(ScalingStateManager.SCALING_STATE_FAILED);
					history.setEndTime(endTime);
					history.setInstances(newCount);
					history.setErrorCode(errorCode);
					historyStore.saveScalingHistory(history); // update history
					appState.setHistoryId(actionId);
				} else {
					logger.error( "Failed to save scaling fail history for app " + appId);
				}
				appState.setInstanceCountState(SCALING_STATE_FAILED);
				appState.setLastActionTriggerId(thresholdType);
				appState.setLastActionInstanceTarget(newCount);
				appState.setLastActionStartTime(startTime);
				appState.setErrorCode(errorCode);
				appState.setScaleEvent(null);
				dataStore.saveScalingState(appState);
				return;
			
			} catch (DataStoreException e) {
				logger.error("Failed to save scaling history for application " + appId, e);
				
			}
			try {
				Thread.sleep(1000);
			} catch (InterruptedException e) {
			}
		

	}
	
	/**
	 * Updates app scaling state
	 * @param org
	 * @param space
	 * @param appName
	 * @throws Exception 
	 */
	public void setScalingStateCompleted(String appId, String actionId) throws Exception{
		long endTime = System.currentTimeMillis();

		try {
			AppAutoScaleState appState = dataStore.getScalingState(appId);
			if (appState == null) {
				logger.error("Failed to save scaling history for app " + appId + " since appState is null.");
				return;
			}

			// Scaling history
			ScalingHistory history = historyStore.getScalingHistory(actionId);

			if (history != null && history.getStatus() == SCALING_STATE_FAILED) {
				return;
			}

			if (history == null) {
				history = appState.getScaleEvent();
			}

			if (history != null) {
				if (!actionId.equals(history.getId())) {
					return;
				}
				history.setStatus(ScalingStateManager.SCALING_STATE_COMPLETED);
				history.setEndTime(endTime);
				historyStore.saveScalingHistory(history); // update history
				appState.setHistoryId(actionId);

			} else {
				logger.error("Failed to save scaling complete history for app " + appId);
			}
			appState.setInstanceCountState(ScalingStateManager.SCALING_STATE_COMPLETED);
			appState.setLastActionEndTime(endTime);
			appState.setScaleEvent(null);
			dataStore.saveScalingState(appState);
			return;
		} catch (DataStoreException e) {

			logger.error("Error occurs when store the state of application " + appId + "." + e.getMessage(), e);
		}
		try {
			Thread.sleep(1000);
		} catch (InterruptedException e) {
		}
		
	}
	
	
	public void correctStateOnStart(String appId) {
		AppAutoScaleState appState = dataStore.getScalingState(appId);
		ScalingHistory history = null;
		if (appState != null) {
			history = appState.getScaleEvent();
		}

		try {
			if (history != null) {
				long currentTime = System.currentTimeMillis();
				CloudApplicationManager manager = CloudApplicationManager
						.getInstance();
				int instances = manager.getInstances(appId);
				if (instances != history.getInstances()) {

					String errorCode = Constants.CloudFoundryInternalError;
					history.setStatus(ScalingStateManager.SCALING_STATE_FAILED);
					history.setErrorCode(errorCode);
					history.setInstances(instances);
					history.setEndTime(history.getStartTime() + 5L);

					historyStore.saveScalingHistory(history); // update history
					appState.setHistoryId(history.getId());

					appState.setInstanceCountState(SCALING_STATE_FAILED);
					appState.setLastActionInstanceTarget(instances);
					appState.setLastActionStartTime(currentTime);
					appState.setErrorCode(errorCode);
					appState.setScaleEvent(null);
					dataStore.saveScalingState(appState);
				} else {
					history.setStatus(ScalingStateManager.SCALING_STATE_COMPLETED);
					history.setEndTime(history.getStartTime() + 20L);
					historyStore.saveScalingHistory(history); // update history
					appState.setHistoryId(history.getId());
					
					appState.setInstanceCountState(ScalingStateManager.SCALING_STATE_COMPLETED);
					appState.setLastActionEndTime(currentTime);
					appState.setScaleEvent(null);
					dataStore.saveScalingState(appState);
				}
			}
		} catch (Exception e) {
			logger.error("Error occurs when store the state of application "
					+ appId + "." + e.getMessage(), e);
		}

	}

	
	/**
	 * Builds ScalingHistory object
	 * @param status
	 * @param startTime
	 * @param endTime
	 * @param thresholdType
	 * @param instances
	 * @param trigger
	 * @return
	 */
	private ScalingHistory buildScalingHistory(String appId, int status, long startTime,long endTime,
			String thresholdType, int currentInstances, int newCount, 
			AutoScalerPolicyTrigger trigger, int triggerType, String errorCode, String scheduleType,String timeZone,Long scheduleStartTime, Integer dayOfWeek) {
		int threshold = 0;
		int adjustment = newCount - currentInstances;
		String metricName = null;
		int breachDuration = 0;
		if (trigger != null){
		    metricName = trigger.getMetricType();
			if (thresholdType.equals(AutoScalerPolicyTrigger.TriggerId_LowerThreshold)){
				threshold = trigger.getLowerThreshold();
			}
			else{
				threshold = trigger.getUpperThreshold();
			}
			breachDuration = trigger.getBreachDuration();
		}
		ScalingHistory history = new ScalingHistory(appId,
				status,  adjustment, newCount, startTime, endTime, metricName, threshold, 
				thresholdType, breachDuration, triggerType, errorCode,scheduleType,timeZone,scheduleStartTime,dayOfWeek);
		return history;
	}
}
