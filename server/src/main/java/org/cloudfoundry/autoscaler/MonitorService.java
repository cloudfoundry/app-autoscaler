package org.cloudfoundry.autoscaler;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.bean.Trigger;
import org.cloudfoundry.autoscaler.exceptions.MonitorServiceException;
import org.cloudfoundry.autoscaler.exceptions.TriggerNotFoundException;
import org.cloudfoundry.autoscaler.exceptions.TriggerNotSubscribedException;
import org.cloudfoundry.autoscaler.metric.MonitorController;


public class MonitorService
{
	private static final Logger logger     = Logger.getLogger(MonitorService.class.getName());

	
	public MonitorService(String appId) throws MonitorServiceException
	{
		
	}
	
	public boolean subscribe(Trigger trigger) throws MonitorServiceException
	{
		try {
			logger.info("Add triggers for application " + trigger.getAppId());
			MonitorController.getInstance().addTrigger(trigger);
		} catch (Exception e) {
			throw new MonitorServiceException("Failed to subscribe triggers", e);
		}
		return true;
	}
	
	public boolean unsubscribe( String appId) throws MonitorServiceException, TriggerNotSubscribedException
	{

		logger.info("Remove triggers of application " + appId);
		try {
			logger.info("Remove triggers of application " + appId);
			MonitorController.getInstance().removeTrigger(appId);
		} catch (TriggerNotFoundException e) {
			throw new TriggerNotSubscribedException("Triggers are not found for application " + appId);
		}
		return true;
	}
	
	
}
