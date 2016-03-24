package org.cloudfoundry.autoscaler.schedule;

import java.text.ParseException;
import java.util.List;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

import org.cloudfoundry.autoscaler.data.couchdb.document.AutoScalerPolicy;
import org.cloudfoundry.autoscaler.manager.PolicyManager;
import org.cloudfoundry.autoscaler.manager.PolicyManagerImpl;

public class DefaultScheduledService implements ScheduledService{

	private static ScheduledService service = new DefaultScheduledService();
	private ScheduledExecutorService executor = Executors.newSingleThreadScheduledExecutor(); 
	
	public static ScheduledService getInstance() {
		return service;
	}
	
	private DefaultScheduledService() {
	}
	
	@Override
	public void registerSchedule(AutoScalerPolicy scalerPolicy) throws ParseException {
	}
	
	@Override
	public void removeSchedule(String appId) {
	}
	
	static class ScheduledDaemon implements Runnable {

		PolicyManager policyManager = PolicyManagerImpl.getInstance();
		@Override
		public void run() {
			List<AutoScalerPolicy> monitoredCache = policyManager.getMonitoredCache();
			for (AutoScalerPolicy policy : monitoredCache) {
				try {
					ScheduledServiceUtil.updateScheduledPolicyBasedOnTime(policy);
				} catch (Throwable e) {
				}
			}
		}
	}

	@Override
	public void start() {
		executor.scheduleWithFixedDelay(new ScheduledDaemon(), 0, 60, TimeUnit.SECONDS);
	}

	@Override
	public void shutdown() {
		executor.shutdown();
	}
}
