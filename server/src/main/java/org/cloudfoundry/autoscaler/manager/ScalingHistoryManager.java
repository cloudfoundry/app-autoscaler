package org.cloudfoundry.autoscaler.manager;

import java.util.LinkedList;
import java.util.List;
import java.util.TimeZone;

import org.cloudfoundry.autoscaler.common.util.TimeZoneUtil;
import org.cloudfoundry.autoscaler.data.AutoScalingDataStore;
import org.cloudfoundry.autoscaler.data.couchdb.AutoScalingDataStoreFactory;
import org.cloudfoundry.autoscaler.data.couchdb.document.AppAutoScaleState;
import org.cloudfoundry.autoscaler.data.couchdb.document.ScalingHistory;
import org.cloudfoundry.autoscaler.exceptions.DataStoreException;

/**
 * This class is used to persist and retrieve scaling history
 * 
 *
 */
public class ScalingHistoryManager {
    public static final ScalingHistoryManager instance = new ScalingHistoryManager();
	private ScalingHistoryManager(){
		
	}
	public static ScalingHistoryManager getInstance(){
		return instance;
	}
	/**
	 * Persist scaling history
	 * @throws DataStoreException 
	 */
	public void saveScalingHistory(ScalingHistory history) throws DataStoreException{
		AutoScalingDataStore dataStore = AutoScalingDataStoreFactory.getAutoScalingDataStore();
		dataStore.saveScalingHistory(history);
	}
	
	
	
	public ScalingHistory getScalingHistory(String id) throws DataStoreException{
		AutoScalingDataStore dataStore = AutoScalingDataStoreFactory.getAutoScalingDataStore();
		return dataStore.getHistoryById(id);
	}
	
	/**
	 * Get scaling history list
	 * @return
	 * @throws DataStoreException 
	 */
	public List<ScalingHistory> getHistoryList(ScalingHistoryFilter filter, String zone) throws DataStoreException{
		AutoScalingDataStore dataStore = AutoScalingDataStoreFactory.getAutoScalingDataStore();
		ScalingHistory activeScaling = null;

		List<ScalingHistory> scalingHistory = dataStore.getHistoryList(filter);
		if (scalingHistory == null){
			scalingHistory = new LinkedList<ScalingHistory>();
		}

		boolean validActiveScaling = false;
		int offset = filter.getOffset();
		if (offset == 0) { // Retrieve the history that is in scaling status
			String status = filter.getStatus();
			/** Check scaling activity **/
			if (status == null
					|| (Integer.parseInt(status) == ScalingStateManager.SCALING_STATE_REALIZING)) {
				AppAutoScaleState scalingState = dataStore
						.getScalingState(filter.getAppId());
				if (scalingState != null) {
					activeScaling = scalingState.getScaleEvent();
					if (activeScaling != null)  {
						if ( ( filter.getScaleType() == null) || (filter.getScaleType().equalsIgnoreCase(filter.SCALE_IN_TYPE) && activeScaling.getAdjustment() <0 )  ||
								(filter.getScaleType().equalsIgnoreCase(filter.SCALE_OUT_TYPE) && activeScaling.getAdjustment() >0 )  ) {
										filter.setOffset(offset - 1);
										filter.setMaxCount(filter.getMaxCount() - 1);
										validActiveScaling = true;
							}
					}
				}
			}
		}
		TimeZone curTimeZone = TimeZone.getDefault();
		TimeZone policyTimeZone = TimeZone.getDefault();
		int timeZoneRawOffSet = curTimeZone.getRawOffset();
		if(null != scalingHistory && scalingHistory.size() > 0)
		{
			String timezone = "";
			if(zone == null || "".equals(zone) || "null".equals(zone))
			{
				ScalingHistory history = scalingHistory.get(scalingHistory.size()-1);
				 timezone= history.getTimeZone();
			}
			else
			{
				timezone = zone;
			}
			policyTimeZone = TimeZoneUtil.parseTimeZoneId(timezone);
			for(ScalingHistory his : scalingHistory)
			{
				long startTime = his.getStartTime();
				his.setRawOffset(timeZoneRawOffSet + policyTimeZone.getOffset(startTime) - curTimeZone.getOffset(startTime));
			}
		}
		if (validActiveScaling)
		{
			ScalingHistory ing = activeScaling;
			String timezone = "";
			if(zone == null || "".equals(zone) || "null".equals(zone))
			{
				 timezone= ing.getTimeZone();
			}
			else
			{
				timezone = zone;
			}
			policyTimeZone = TimeZoneUtil.parseTimeZoneId(timezone);
			long startTime = ing.getStartTime();
			ing.setRawOffset(timeZoneRawOffSet + policyTimeZone.getOffset(startTime) - curTimeZone.getOffset(startTime));
		}

		if (validActiveScaling)
			scalingHistory.add(0, activeScaling);
		return scalingHistory;
	}
	
	/**
	 * Get scaling history count
	 * @param filter
	 * @return
	 * @throws DataStoreException
	 */
	public int getHistoryCount(ScalingHistoryFilter filter)
			throws DataStoreException {
		AutoScalingDataStore dataStore = AutoScalingDataStoreFactory
				.getAutoScalingDataStore();
		int count = dataStore.getHistoryCount(filter);
		String status = filter.getStatus();
		/** Check scaling activity **/
		if (status == null || (Integer.parseInt(status) == ScalingStateManager.SCALING_STATE_REALIZING)) {
			AppAutoScaleState scalingState = dataStore.getScalingState(filter
					.getAppId());
			if (scalingState != null && scalingState.getScaleEvent() != null){
				ScalingHistory activeScaling = scalingState.getScaleEvent();
				if (activeScaling != null) {
						if ( ( filter.getScaleType() == null) || (filter.getScaleType().equalsIgnoreCase(filter.SCALE_IN_TYPE) && activeScaling.getAdjustment() <0 )  ||
								(filter.getScaleType().equalsIgnoreCase(filter.SCALE_OUT_TYPE) && activeScaling.getAdjustment() >0 )  ) {
							count ++;
						}
				}
			}
		}
		return count;
	}
}
