package org.cloudfoundry.autoscaler.servicebroker;

import javax.servlet.ServletContextEvent;
import javax.servlet.ServletContextListener;
import javax.servlet.annotation.WebListener;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.servicebroker.mgr.ScalingServiceMgr;
import org.cloudfoundry.autoscaler.servicebroker.util.DataSourceUtil;

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
	 *
	 * @see ServletContextListener#contextInitialized(ServletContextEvent)
	 */
	public void contextInitialized(ServletContextEvent event) {
		try {
			DataSourceUtil.setStoreProvider(Constants.CONFIG_ENTRY_DATASTORE_PROVIDER_COUCHDB);
			ScalingServiceMgr.getInstance();
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
			throw new RuntimeException(e);
		}
	}

	/**
	 * shutdown all thread pools when the server goes down
	 *
	 * @see ServletContextListener#contextDestroyed(ServletContextEvent)
	 */
	public void contextDestroyed(ServletContextEvent event) {
		logger.info("Finished to shutdown all thread pools.");
	}

}
