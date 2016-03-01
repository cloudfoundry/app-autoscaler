package org.cloudfoundry.autoscaler.metric.poller;

import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;

public class CFPollerManager {
    public ConcurrentMap<String, CFAppStatsPoller> appStatsPollerMap = new ConcurrentHashMap<String, CFAppStatsPoller>();
    public ConcurrentMap<String, CFAppInfoPoller> appInfoPollerMap = new ConcurrentHashMap<String, CFAppInfoPoller>();
    private static CFPollerManager manager = new CFPollerManager();
    private volatile boolean stopped = false;

    private CFPollerManager() {

    }

    public static CFPollerManager getInstance() {
        return manager;
    }

    public void shutdown() {
        stopped = true;
        CFAppStatsPoller.shutdown();

    }

    public void addAppInfoPoller(String appId) {
        if (stopped) 
        	return;
        CFAppInfoPoller poller = appInfoPollerMap.get(appId);
        if (poller == null) {
            poller = new CFAppInfoPoller(appId);
            CFAppInfoPoller olderPoller = appInfoPollerMap.putIfAbsent(appId, poller);
            if (olderPoller != null) {
                poller = olderPoller;
            }
        }
        poller.start();
        
    }
    
    
    public void removeAppInfoPoller(String appId) {
        CFAppInfoPoller poller = appInfoPollerMap.get(appId);
        if (poller != null) {
            poller.stop();
            appInfoPollerMap.remove(appId);

        }
    }
    
    public void addPoller(String appId) {
        if (stopped) 
        	return;
        
        CFAppStatsPoller poller = appStatsPollerMap.get(appId);
        if (poller == null) {
            poller = new CFAppStatsPoller(appId);
            CFAppStatsPoller olderPoller = appStatsPollerMap.putIfAbsent(appId, poller);
            if (olderPoller != null) {
                poller = olderPoller;
            }
        }
        poller.start();
        
    }

    public void removePoller(String appId) {
        CFAppStatsPoller poller = appStatsPollerMap.get(appId);
        if (poller != null) {
            poller.stop();
            appStatsPollerMap.remove(appId);

        }
    }
 
}
