package org.cloudfoundry.autoscaler.scheduler.util;

import org.quartz.JobExecutionContext;
import org.quartz.JobExecutionException;
import org.quartz.listeners.JobListenerSupport;

public class TestJobListener extends JobListenerSupport {
	private int expectedNumOfJobFired;
	private int currentNumOfFire = 0;


	public TestJobListener(int expectedNumOfJobFired) {
		this.expectedNumOfJobFired = expectedNumOfJobFired;
	}

	@Override
	public String getName() {
		return "default";
	}

	@Override
	public void jobWasExecuted(JobExecutionContext context, JobExecutionException jobException) {
		currentNumOfFire++;

		if (currentNumOfFire < expectedNumOfJobFired) {
			return;
		}

		synchronized (this) {
			notify();
		}
	}

	synchronized public void waitForJobToFinish(long timeoutMillis) throws InterruptedException {
		wait(timeoutMillis);
		if (currentNumOfFire < expectedNumOfJobFired) {
			throw new RuntimeException("Waiting for job time out. Number of times job fired: " + currentNumOfFire
					+ " expected: " + expectedNumOfJobFired);
		}
	}
}
