package org.cloudfoundry.autoscaler.scheduler.util;

import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationSchedules;
import org.cloudfoundry.autoscaler.scheduler.rest.model.Schedules;

public class ApplicationPolicyBuilder {
	
	private ApplicationSchedules applicationPolicy;
	
	public ApplicationPolicyBuilder(int instance_min_count, int instance_max_count, String timezone,
			int noOfSpecificDateSchedules, int noOfDOMRecurringSchedules, int noOfDOWRecurringSchedules) {
		applicationPolicy = new ApplicationSchedules();
		applicationPolicy.setInstance_min_count(instance_min_count);
		applicationPolicy.setInstance_max_count(instance_max_count);
		applicationPolicy.setSchedules(new ScheduleBuilder(timezone, noOfSpecificDateSchedules, noOfDOMRecurringSchedules, noOfDOWRecurringSchedules).build());

	}

	public ApplicationPolicyBuilder setSchedules(Schedules schedules){
		this.applicationPolicy.setSchedules(schedules);
		return this;
	}
	
	public ApplicationSchedules build(){
		return applicationPolicy;
	}
	
}
