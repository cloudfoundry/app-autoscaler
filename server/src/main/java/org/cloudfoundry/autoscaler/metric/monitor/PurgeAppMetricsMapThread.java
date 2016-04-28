package org.cloudfoundry.autoscaler.metric.monitor;

import java.util.Map.Entry;
import java.util.Set;
import java.util.concurrent.ConcurrentMap;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.bean.InstanceMetrics;
import org.cloudfoundry.autoscaler.common.util.CloudFoundryManager;
import org.cloudfoundry.autoscaler.common.util.ConfigManager;
import org.cloudfoundry.autoscaler.constant.Constants;
import org.cloudfoundry.autoscaler.metric.bean.ApplicationMetrics;

public class PurgeAppMetricsMapThread implements Runnable {
    private static final Logger logger = Logger.getLogger(PurgeAppMetricsMapThread.class);
    private ConcurrentMap<String, ApplicationMetrics> appMetricsMap;

    public PurgeAppMetricsMapThread(ConcurrentMap<String, ApplicationMetrics> appMetricsMap) {
        this.appMetricsMap = appMetricsMap;
    }

    @Override
    public void run() {
        Set<Entry<String, ApplicationMetrics>> entries = appMetricsMap.entrySet();
        long now = System.currentTimeMillis();
        for (Entry<String, ApplicationMetrics> entry : entries) {
            String appId = entry.getKey();
            ApplicationMetrics appMetrics = entry.getValue();
            long reportInterval = ConfigManager.getInt(Constants.REPORT_INTERVAL, 120);
            long timeout = (long) (reportInterval * 1.5 * 1000 );
            // purge app
            long appTimestamp = appMetrics.getTimestamp();
            if (now - appTimestamp > timeout) {
            	String[] appInfo = null;
        		try {
        			appInfo = CloudFoundryManager.getInstance().getAppInfoByAppId(appId);
        		} catch (Exception e) {
					logger.error(String.format(
							"Failed to get the state for app %s with exception %s", appId, e.getMessage()));
        		}

        		// if app the stopped, then remove it from "appMetrics" map.
    			if (appInfo[3].equalsIgnoreCase(Constants.CF_APPLICATION_STATE_STOPPED)) {
    				logger.info("remove app " + appMetrics.getAppId() + ", stopped & last upate at " + appTimestamp);
    				appMetricsMap.remove(appId);
    			}	else if (appMetrics.getPollerMetricsMap().size() == 0){
    				//if the app is started, but no instance metric at all (maybe all the instances are crashed).
    				logger.info("remove app " + appMetrics.getAppId() + ", no instances metric available since " + appTimestamp);
    				appMetricsMap.remove(appId);
    			}  else {
    				//do nothing
    			}
    			

            } else {// purge instance

            	int runningInstances = -1; 
                
               
                
                ConcurrentMap<Integer, InstanceMetrics> metrics = appMetrics.getPollerMetricsMap();
                Set<Entry<Integer, InstanceMetrics>> metricsEntries = metrics.entrySet();
                for (Entry<Integer, InstanceMetrics> metricsEntry : metricsEntries) {
                    int index = metricsEntry.getKey();
                    InstanceMetrics m = metricsEntry.getValue();
                    if (now - m.getTimestamp() > timeout) {
                    	if (runningInstances == -1 ){
                    		String[] appInfo = null;
                    		try {
                    			appInfo = CloudFoundryManager.getInstance().getAppInfoByAppId(appId);
                    			runningInstances  = Integer.parseInt(appInfo[4]);
                    		} catch (Exception e) {
            					logger.error(String.format(
            							"Failed to get the state for app %s with exception %s", appId, e.getMessage()));
                    		}
                    	}

                    	//instances number is smaller than the recorded instance index, i.e. instances number =3 while the currnet index is 3 as well.
						if (runningInstances <= index ) { 
	                        metrics.remove(index);
	                        logger.info("remove CF instance " + index + " for app " + appMetrics.getAppId() + ", due to timeout(ms) in " + timeout);
						}
					}
                }
            }
        }
    }
}
