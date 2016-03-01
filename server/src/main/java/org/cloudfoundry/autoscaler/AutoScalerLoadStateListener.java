package org.cloudfoundry.autoscaler;



import javax.servlet.ServletContextEvent;
import javax.servlet.ServletContextListener;
import javax.servlet.annotation.WebListener;

import org.apache.log4j.Logger;


@WebListener
public class AutoScalerLoadStateListener implements ServletContextListener
{
	private static final String CLASS_NAME = AutoScalerLoadStateListener.class.getName();
	private static final Logger logger     = Logger.getLogger(CLASS_NAME); 

    @Override
    public void contextInitialized(ServletContextEvent event)
    {
        logger.info("AutoScaler servlet created");
        
    }

    @Override
    public void contextDestroyed(ServletContextEvent event)
    {
        logger.info("AutoScaler servlet destroyed");
    }
	
}
