package org.cloudfoundry.autoscaler.metric.listener;

import java.util.List;
import java.util.Map;
import java.util.Map.Entry;
import java.util.Set;

import javax.servlet.ServletContextEvent;
import javax.servlet.ServletContextListener;
import javax.servlet.annotation.WebListener;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.ScalingScheduledServiceFactory;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.BoundApp;
import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.TriggerRecord;
import org.cloudfoundry.autoscaler.event.ScalingStateManager;
import org.cloudfoundry.autoscaler.metric.MonitorController;
import org.cloudfoundry.autoscaler.metric.poller.CFPollerManager;
import org.cloudfoundry.autoscaler.policy.PolicyManager;
import org.cloudfoundry.autoscaler.policy.PolicyManagerImpl;
import org.cloudfoundry.autoscaler.statestore.AutoScalingDataStore;
import org.cloudfoundry.autoscaler.statestore.AutoScalingDataStoreFactory;

/**
 * Application Lifecycle Listener implementation class LifecycleListener
 * 
 */
@WebListener
public class LifecycleListener implements ServletContextListener {
    private static final Logger logger = Logger.getLogger(LifecycleListener.class);

    public LifecycleListener() {
        logger.info("LifecycleListener initialized.");
    }

    /**
     * Load registered triggers from mongodb store
     * 
     * @see ServletContextListener#contextInitialized(ServletContextEvent)
     */
    public void contextInitialized(ServletContextEvent event) {
        try {
            loadServiceBindings();
        	loadTriggers();
            loadScheduledCache();
            ScalingScheduledServiceFactory.getScheduledService().start();
        } catch (Exception e) {
            logger.error(e.getMessage(), e);
        }
    }

    /**
     * shutdown all thread pools when the server goes down
     * 
     * @see ServletContextListener#contextDestroyed(ServletContextEvent)
     */
    public void contextDestroyed(ServletContextEvent event) {
        CFPollerManager.getInstance().shutdown();
        MonitorController.getInstance().shutdown();
        ScalingScheduledServiceFactory.getScheduledService().shutdown();
        logger.info("Finished to shutdown all thread pools.");
    }
    
    private void loadTriggers() throws Exception {
        AutoScalingDataStore storeService = AutoScalingDataStoreFactory
				.getAutoScalingDataStore();
        Map<String, List<TriggerRecord>> triggersMap = storeService.getAllTriggers();
        Set<String> keys = triggersMap.keySet();
        if (keys.size() == 0) {
            return;
        }
        MonitorController controller = MonitorController.getInstance();
        for (String key : keys) {
            List<TriggerRecord> triggers = triggersMap.get(key);
            for (TriggerRecord record : triggers) {
                controller.addTriggerDirectly(record.getTrigger());
                logger.info("added " + record);
            }
        }
    }

    private void loadServiceBindings() throws Exception {
         AutoScalingDataStore storeService = AutoScalingDataStoreFactory
				.getAutoScalingDataStore();
        MonitorController controller = MonitorController.getInstance();
        Set<Entry<String, List<BoundApp>>> bindingsSet = storeService.getAllBindings().entrySet();
        for (Entry<String, List<BoundApp>> entry : bindingsSet) {
            String serviceId = entry.getKey();
            List<BoundApp> boundApps = entry.getValue();
            for (BoundApp boundApp : boundApps) {
                String appId = boundApp.getAppId();
                controller.addOrUpdateBoundApp(serviceId, appId, boundApp.getAppType(), boundApp.getAppName());
                controller.addPoller(appId);
                ScalingStateManager.getInstance().correctStateOnStart(appId);
            }
        }
    }
    
    private void loadScheduledCache() throws Exception {
		PolicyManager policyManager = PolicyManagerImpl.getInstance();
		policyManager.recoverMonitoredCache();
    }

}
