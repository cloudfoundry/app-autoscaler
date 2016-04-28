package org.cloudfoundry.autoscaler.manager;

import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.List;
import java.util.TimeZone;
import java.util.UUID;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.bean.AutoScalerPolicyTrigger;
import org.cloudfoundry.autoscaler.bean.MonitorTriggerEvent;
import org.cloudfoundry.autoscaler.common.util.CloudFoundryManager;
import org.cloudfoundry.autoscaler.common.util.ConfigManager;
import org.cloudfoundry.autoscaler.constant.Constants;
import org.cloudfoundry.autoscaler.data.AutoScalingDataStore;
import org.cloudfoundry.autoscaler.data.couchdb.AutoScalingDataStoreFactory;
import org.cloudfoundry.autoscaler.data.couchdb.document.AppAutoScaleState;
import org.cloudfoundry.autoscaler.data.couchdb.document.Application;
import org.cloudfoundry.autoscaler.data.couchdb.document.AutoScalerPolicy;
import org.cloudfoundry.autoscaler.data.couchdb.document.ScheduledPolicy;
import org.cloudfoundry.autoscaler.data.couchdb.document.ScheduledPolicy.ScheduledType;
import org.cloudfoundry.autoscaler.exceptions.AppNotFoundException;
import org.cloudfoundry.autoscaler.exceptions.CloudException;
import org.cloudfoundry.autoscaler.exceptions.DataStoreException;
import org.cloudfoundry.autoscaler.exceptions.NoAttachedPolicyException;
import org.cloudfoundry.autoscaler.exceptions.PolicyNotFoundException;
import org.cloudfoundry.autoscaler.schedule.ScheduledServiceUtil;
import org.cloudfoundry.autoscaler.util.IcapMonitorMetricsMapper;

public class ApplicationScaleManagerImpl implements ApplicationScaleManager {
	private static ApplicationScaleManagerImpl instance = new ApplicationScaleManagerImpl();

	private ApplicationScaleManagerImpl() {
	};

	public static ApplicationScaleManagerImpl getInstance() {
		return instance;
	}

	private static final Logger logger = Logger.getLogger(ApplicationScaleManager.class.getName());
	private static final Logger loggerEvent = Logger.getLogger("triggerevent");
	public final long eventTimeout = Long.parseLong(ConfigManager.get("LAST_TRIGGER_EVENT_TIME_OUT", "10")) * 60
			* 1000L;

	@Override
	public void doScaleByTrigger(MonitorTriggerEvent triggerEvent) throws Exception {

		String appId = triggerEvent.getAppId();
		String triggerId = triggerEvent.getTriggerId();
		AutoScalerPolicy policy = null;
		AutoScalerPolicyTrigger policyTrigger = null;

		logger.debug("Receive trigger " + triggerEvent.toString() + "with handler " + this.toString() + " for appId = "
				+ appId + ", triggerId = " + triggerId + ", metricType = " + triggerEvent.getMetricType());
		loggerEvent.debug(
				"Receive trigger " + triggerEvent.toString() + "with handler " + this.toString() + " for appId = "
						+ appId + ", triggerId = " + triggerId + ", metricType = " + triggerEvent.getMetricType());
		int currentInstanceCount = this.getCurrentInstanceCount(appId);
		try {
			policy = getPolicy(triggerEvent.getAppId());
			policyTrigger = getAutoScalerPolicyTrigger(policy, triggerEvent);
			if (policyTrigger == null) {
				logger.warn("PolicyTrigger is null.");
				return;
			}

			if (!validateCooldownSetting(appId, policyTrigger, triggerId)) {
				logger.debug("Abort trigger " + triggerEvent.toString() + " for event " + triggerEvent.getMetricType()
						+ "/" + triggerId + " as a scaling action is ongoing or cooldown time " + appId);
				return;
			}

			int newCount = 0;
			if (validateInstanceCounts(appId, policy, triggerId, currentInstanceCount)) {
				newCount = calculateNewCount(policy, policyTrigger, triggerId, currentInstanceCount);
				logger.info("Handle monitor trigger" + this.toString() + " : appId = " + appId + ", triggerId = "
						+ triggerId + ", metricValue = " + triggerEvent.getMetricValue());
				logger.info("Scale: Target instance count for app " + appId + " is " + newCount);
				this.scale(appId, currentInstanceCount, newCount, null, policy.getTimezone(), null, null);
			} else {
				logger.debug("Abort trigger " + triggerEvent.toString() + " for event " + triggerEvent.getMetricType()
						+ "/" + triggerId + " as it reachs the max/min instance count. " + appId);
				return;
			}

		} catch (PolicyNotFoundException e) {
			logger.error("The policy for app " + appId + " .can not be found.", e);
			return;
		} catch (DataStoreException e) {
			logger.error("Error occurs when getting the policy for app: " + appId, e);
			return;
		} catch (CloudException e) {
			logger.error("Failed to get find the application ", e);
		} catch (Exception e) {
			logger.error("Error occurs when handle trigger event for  app: " + appId, e);
			return;
		}

	}

	@Override
	public void doScaleBySchedule(String appId, AutoScalerPolicy policy) throws Exception {
		Long startTime = this.getStartTime(policy);
		Integer dayOfWeek = this.getDayOfWeek(policy);
		int currentInstanceCount = this.getCurrentInstanceCount(appId);
		if (this.validateInstanceCounts(appId, policy, null, currentInstanceCount)) {

			int newCount = this.calculateNewCount(policy, null, null, currentInstanceCount);
			this.scale(appId, currentInstanceCount, newCount, policy.getCurrentScheduleType(), policy.getTimezone(),
					startTime, dayOfWeek);
		}
	}

	private Integer getDayOfWeek(AutoScalerPolicy policy) {
		Integer dayOfWeek = null;
		String scheduleType = policy.getCurrentScheduleType();
		if (ScheduledType.RECURRING.name().equals(scheduleType)) {
			dayOfWeek = ScheduledServiceUtil.dayOfWeek(new Date());
		}
		return dayOfWeek;
	}

	private Long getStartTime(AutoScalerPolicy policy) {
		String scheduleType = policy.getCurrentScheduleType();
		String timeZone = policy.getTimezone();
		String startTimeStr = policy.getCurrentScheduleStartTime();
		Long startTime = 0l;

		if (null != startTimeStr && !"".equals(startTimeStr) && null != scheduleType && !"".equals(scheduleType)) {
			if (ScheduledType.RECURRING.name().equals(scheduleType)) {
				try {
					startTime = new SimpleDateFormat(ScheduledPolicy.recurringDateFormat).parse(startTimeStr).getTime();
				} catch (ParseException e) {
					// TODO Auto-generated catch block
					e.printStackTrace();
				}

			} else if (ScheduledType.SPECIALDATE.name().equals(scheduleType)) {
				try {
					startTime = new SimpleDateFormat(ScheduledPolicy.specialDateDateFormat).parse(startTimeStr)
							.getTime();
				} catch (ParseException e) {
					// TODO Auto-generated catch block
					e.printStackTrace();
				}
			}

			TimeZone curTimeZone = TimeZone.getDefault();
			TimeZone policyTimeZone = TimeZone.getDefault();
			String zoneName = "";
			if (null != timeZone && !"".equals(timeZone)) {
				timeZone = timeZone.trim().replaceAll("\\s+", "");
				int index1 = timeZone.indexOf("(");
				int index2 = timeZone.indexOf(")");
				if (index2 >= 0) {
					zoneName = timeZone.substring(index2 + 1, timeZone.length()).trim();
				} else {
					if (index2 > index1) {
						zoneName = timeZone.substring(index1 + 1, index2);

					}
				}
				policyTimeZone = TimeZone.getTimeZone(zoneName);
			}
			startTime = startTime - policyTimeZone.getRawOffset() + curTimeZone.getRawOffset();

		}
		return startTime;
	}

	private AutoScalerPolicyTrigger getAutoScalerPolicyTrigger(AutoScalerPolicy policy, MonitorTriggerEvent event)
			throws AppNotFoundException, PolicyNotFoundException {
		List<AutoScalerPolicyTrigger> triggers = policy.getPolicyTriggers();
		if (triggers == null)
			throw new PolicyNotFoundException("No policy for metric: " + event.getMetricType());
		for (AutoScalerPolicyTrigger trigger : triggers) {
			String metric = IcapMonitorMetricsMapper.getMetricNameMapper().get(trigger.getMetricType().toUpperCase());
			if (metric.equalsIgnoreCase(event.getMetricType())) {
				return trigger;
			}
		}
		throw new PolicyNotFoundException("No policy for metric: " + event.getMetricType());
	}

	private int calculateNewCount(AutoScalerPolicy policy, AutoScalerPolicyTrigger policyTrigger, String triggerId,
			int currentInstanceCount) {
		int newCount = 0;
		if (null != policyTrigger) {
			int instanceStep = policyTrigger.getInstanceStepCountUp();
			String adjustmentType = null;
			if (AutoScalerPolicyTrigger.TriggerId_LowerThreshold.equals(triggerId)) {
				instanceStep = policyTrigger.getInstanceStepCountDown(); // is a negative value
				adjustmentType = policyTrigger.getScaleInAdjustmentType();
			} else
				adjustmentType = policyTrigger.getScaleOutAdjustmentType();
			if (AutoScalerPolicyTrigger.ADJUSTMENT_CHANGE_PERCENTAGE.equals(adjustmentType)) {
				int adjustment = instanceStep * currentInstanceCount / 100;
				if (adjustment == 0) {
					if (instanceStep < 0)
						adjustment = -1;
					else
						adjustment = 1;
				}
				newCount = currentInstanceCount + adjustment;
			} else
				newCount = currentInstanceCount + instanceStep;
		}

		if (newCount < policy.getCurrentInstanceMinCount()) {
			newCount = policy.getCurrentInstanceMinCount();
		} else if (newCount > policy.getCurrentInstanceMaxCount()) {
			newCount = policy.getCurrentInstanceMaxCount();
		}
		return newCount;
	}

	/*****************************************************************************************************************
	 * 
	 * @throws PolicyNotFoundException
	 * @throws DataStoreException
	 * @throws NoAttachedPolicyException
	 * @throws CloudException
	 * @throws AppNotFoundException
	 */
	public AutoScalerPolicy getPolicy(String appId)
			throws PolicyNotFoundException, DataStoreException, CloudException, AppNotFoundException {
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

	private long getCooldownSecs(AutoScalerPolicyTrigger policyTrigger, String triggerId) {
		if (AutoScalerPolicyTrigger.TriggerId_LowerThreshold.equals(triggerId)) {
			return policyTrigger.getStepDownCoolDownSecs();
		} else
			return policyTrigger.getStepUpCoolDownSecs();
	}

	/**
	 * Checks if the app should be scaled in/out according to cooldown settings
	 * 
	 * @return true if should scale in/out
	 */
	private boolean validateCooldownSetting(String appId, AutoScalerPolicyTrigger policyTrigger, String triggerId) {
		AutoScalingDataStore stateStore = AutoScalingDataStoreFactory.getAutoScalingDataStore();
		AppAutoScaleState appState = null;
		appState = stateStore.getScalingState(appId);

		if (appState == null) {
			// if this is the first trigger for this app we don't
			// have an app-state, so we create the initial one
			return true;
		} else if ((appState.getInstanceCountState() != ScalingStateManager.SCALING_STATE_COMPLETED)
				&& (appState.getInstanceCountState() != ScalingStateManager.SCALING_STATE_FAILED)) {
			long lastStartTime = appState.getLastActionStartTime();
			long currentTime = System.currentTimeMillis();
			boolean timeExpired = (currentTime - lastStartTime) > this.eventTimeout;
			if (timeExpired) {
				logger.debug("True: Last scaling action is not completed but it's time expired for application " + appId
						+ ".");
				return true;
			}
			logger.debug("False: Last scaling action is not completed for application " + appId + ".");
			return false;
		} else {
			/** Check cool down time **/
			long cooldownEndtime = appState.getLastActionEndTime() + 1000L * getCooldownSecs(policyTrigger, triggerId);
			if (System.currentTimeMillis() < cooldownEndtime) {// in cooldown time
				logger.debug("False: It's cooldown time for application " + appId + ". No scaling in action.");
				return false;
			}
		}
		return true;
	}

	/**
	 * Checks if min instance count < app currnent instance count < max instance count
	 * 
	 * @return true if valid to scale in/out
	 * @throws Exception
	 */

	private boolean validateInstanceCounts(String appId, AutoScalerPolicy policy, String triggerId,
			Integer currentInstanceCount) throws Exception {
		int minCount = policy.getCurrentInstanceMinCount();
		int maxCount = policy.getCurrentInstanceMaxCount();
		boolean validScaling = false;
		if (null == triggerId) {
			/** Triggered by the min/max instance definition change in the policy **/
			if (currentInstanceCount < minCount || currentInstanceCount > maxCount) {
				logger.debug(String.format(
						"The current instance count of app %s is %d which beyond the valid instance scope [%d, %d]",
						appId, currentInstanceCount, minCount, maxCount));
				validScaling = true;
			}
		} else {
			/** Triggered by dynamic scale. Check minimum count and maximum count **/
			if (AutoScalerPolicyTrigger.TriggerId_LowerThreshold.equals(triggerId)
					&& currentInstanceCount <= minCount) {
				logger.debug(String.format("Invalid scaling event %s for app %s, since the Min instance count reached",
						triggerId, appId));
			} else if (AutoScalerPolicyTrigger.TriggerId_UpperThreshold.equals(triggerId)
					&& currentInstanceCount >= maxCount) {
				logger.debug(String.format("Invalid scaling event %s for app %s, since the Max instance count reached",
						triggerId, appId));
			} else {
				validScaling = true;
			}
		}

		return validScaling;

	}

	private int getCurrentInstanceCount(String appId) throws Exception {
		int currentInstances = 0;
		try {
			currentInstances = CloudFoundryManager.getInstance().getAppInstancesByAppId(appId);
		} catch (AppNotFoundException e) {
			throw new AppNotFoundException("Application " + appId + " can not be found.");
		} catch (CloudException e) {
			logger.error("Failed to get application instances", e);
			return currentInstances;
		}
		return currentInstances;
	}

	private void scale(String appId, Integer currentInstanceCount, Integer newCount, String scheduleType,
			String timeZone, Long startTime, Integer dayOfWeek) throws Exception {
		ScalingStateManager stateManager = ScalingStateManager.getInstance();
		CloudApplicationManager manager = CloudApplicationManager.getInstance();
		String actionUUID = UUID.randomUUID().toString();
		if (stateManager.setScalingStateRealizing(appId, null, currentInstanceCount, newCount, null,
				TriggerEventHandler.TRIGGER_TYPE_POLICY_CHANGED, actionUUID, scheduleType, timeZone, startTime,
				dayOfWeek)) {
			try {
				// start scaling until update appState successfully
				manager.scaleApplication(appId, newCount);
				ScalingStateMonitorTask task = new ScalingStateMonitorTask(appId, newCount, actionUUID);
				ScalingStateMonitor.getInstance().monitor(task);
			} catch (CloudException e2) {
				String errorCode = e2.getErrorCode();
				if (Constants.MemoryQuotaExceeded.equals(errorCode)) {
					logger.error("Failed to scale application " + appId
							+ ". You have exceeded your organization's memory limit.");
				} else {
					errorCode = Constants.CloudFoundryInternalError;
					logger.error("Failed to scale application " + appId + "." + e2.getMessage());
				}
				stateManager.setScalingStateFailed(appId, null, currentInstanceCount, currentInstanceCount, null,
						TriggerEventHandler.TRIGGER_TYPE_POLICY_CHANGED, errorCode, actionUUID, scheduleType, timeZone,
						startTime, dayOfWeek);

			} catch (Exception e) {
				logger.error("Failed to update application " + appId + " to scaling state." + e.getMessage(), e);
				stateManager.setScalingStateFailed(appId, null, currentInstanceCount, currentInstanceCount, null,
						TriggerEventHandler.TRIGGER_TYPE_POLICY_CHANGED, e.getMessage(), actionUUID, scheduleType,
						timeZone, startTime, dayOfWeek);

				return;
			}
		}

	}

}
