package org.cloudfoundry.autoscaler.schedule;

public class ScalingScheduledServiceFactory {

	public static ScheduledService getScheduledService() {
		return DefaultScheduledService.getInstance();
	}
}
