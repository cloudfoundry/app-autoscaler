package org.cloudfoundry.autoscaler;

import java.text.ParseException;

import org.cloudfoundry.autoscaler.data.couchdb.document.AutoScalerPolicy;



public interface ScheduledService {
	public void registerSchedule(AutoScalerPolicy scalerPolicy) throws ParseException;
	public void removeSchedule(String policyId);
	public void start();
	public void shutdown();
}
