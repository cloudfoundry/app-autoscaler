package org.cloudfoundry.autoscaler.servicebroker.rest;

import java.util.HashSet;
import java.util.Set;

import javax.ws.rs.core.Application;

public class ServiceApplication extends Application {

	@Override
	public Set<Class<?>> getClasses() {
		Set<Class<?>> classes = new HashSet<Class<?>>();
		classes.add(AutoScalingServiceBrokerRest.class);
		return classes;
	}
	
}