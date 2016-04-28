package org.cloudfoundry.autoscaler.manager;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.bean.MonitorTriggerEvent;
import org.cloudfoundry.autoscaler.common.util.ConfigManager;
import org.cloudfoundry.autoscaler.data.couchdb.AutoScalingDataStoreFactory;
import org.cloudfoundry.autoscaler.exceptions.AppNotFoundException;
import org.cloudfoundry.autoscaler.exceptions.CloudException;
import org.cloudfoundry.autoscaler.exceptions.DataStoreException;
import org.cloudfoundry.autoscaler.exceptions.PolicyNotFoundException;

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
	private String appId;
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
		AutoScalingDataStoreFactory.getAutoScalingDataStore();
		ScalingStateManager.getInstance();
		appId = event.getAppId();
		event.getTriggerId();
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
		ApplicationScaleManager manager = ApplicationScaleManagerImpl.getInstance();
		try {
			manager.doScaleByTrigger(event);
		} catch (Exception e) {
			// TODO Auto-generated catch block
			logger.error(e.getMessage());
			logger.error("Error occurs when handle trigger event for  app: " + appId, e);
			e.printStackTrace();
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
}
