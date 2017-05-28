package org.cloudfoundry.autoscaler.scheduler.util;

import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationSchedules;
import org.cloudfoundry.autoscaler.scheduler.rest.model.Schedules;

public class ApplicationPolicyBuilder {
	
	private ApplicationSchedules applicationPolicy;
	public ApplicationPolicyBuilder(int instanceMinCount, int instanceMaxCount){
		applicationPolicy = new ApplicationSchedules();
		applicationPolicy.setInstanceMinCount(instanceMinCount);
		applicationPolicy.setInstanceMaxCount(instanceMaxCount);
	}
	public ApplicationPolicyBuilder(int instanceMinCount, int instanceMaxCount, String timezone,
			int noOfSpecificDateSchedules, int noOfDOMRecurringSchedules, int noOfDOWRecurringSchedules) {
		applicationPolicy = new ApplicationSchedules();
		applicationPolicy.setInstanceMinCount(instanceMinCount);
		applicationPolicy.setInstanceMaxCount(instanceMaxCount);
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
