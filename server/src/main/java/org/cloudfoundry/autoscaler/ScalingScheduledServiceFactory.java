package org.cloudfoundry.autoscaler;

public class ScalingScheduledServiceFactory {

	public static ScheduledService getScheduledService() {
		return DefaultScheduledService.getInstance();
	}
}
