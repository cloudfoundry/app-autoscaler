package org.cloudfoundry.autoscaler.manager;

import java.util.List;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.bean.MonitorTriggerEvent;


public class ScalingEventManager {
    private static final Logger logger = Logger.getLogger(ScalingEventManager.class);
    private static ScalingEventManager instance = new ScalingEventManager();

    private ConcurrentMap<String, TriggerEventHandler> eventHandlerMap = new ConcurrentHashMap<String, TriggerEventHandler>();
    private ConcurrentMap<String, MonitorTriggerEvent> pendingEventMap = new ConcurrentHashMap<String, MonitorTriggerEvent>();
    
    private ScalingEventManager() {
    }
    public static ScalingEventManager getInstance(){
    	return instance;
    }
    
    public boolean addTriggerEvents (MonitorTriggerEvent event){
      	String key = event.getAppId() + ":" + event.getTriggerId();
    	MonitorTriggerEvent pendingEvent = pendingEventMap.get(key);
    	if (pendingEvent == null){
    		pendingEventMap.put(key, event);
    		return true;
    	} else {
			return false;
		}    	
    }
    
    public void processTriggerEvents (MonitorTriggerEvent event){
    	try{ 
    		TriggerEventHandler handler = null;
    		handler = eventHandlerMap.get(event.getAppId());
    		if (handler == null){
    			synchronized (this) {
    				handler = eventHandlerMap.get(event.getAppId());
    				if (handler == null){
    					handler = new TriggerEventHandler();
    					eventHandlerMap.put(event.getAppId(), handler);
    				}
    			}
    		}
    		handler.handleEvent(event);

    	} catch (Exception e) {
    		logger.error("error to post trigger event", e);
    	} finally {
    		String key = event.getAppId() + ":" + event.getTriggerId();
    		pendingEventMap.remove(key);
    	}
    	
    }

    public  boolean postTriggerEvents(List<MonitorTriggerEvent> triggerEventList) {
        try {
            if (triggerEventList == null || triggerEventList.size() == 0) {
                logger.info("No events.");
                return false;
            }
            
            TriggerEventHandler handler = null;
            for (MonitorTriggerEvent event: triggerEventList){
            	handler = eventHandlerMap.get(event.getAppId());
            	if (handler == null){
            		synchronized (this) {
            			handler = eventHandlerMap.get(event.getAppId());
            			if (handler == null){
            				handler = new TriggerEventHandler();
            				eventHandlerMap.put(event.getAppId(), handler);
            			}
            		}
            	}
            	handler.handleEvent(event);

            }

        } catch (Exception e) {
        	logger.error("error to post trigger event", e);
            
        } 
        return false;
    }

}
