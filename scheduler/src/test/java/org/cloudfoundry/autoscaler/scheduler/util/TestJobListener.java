package org.cloudfoundry.autoscaler.scheduler.util;

import java.util.UUID;
import java.util.concurrent.atomic.AtomicInteger;
import org.quartz.JobExecutionContext;
import org.quartz.JobExecutionException;
import org.quartz.listeners.JobListenerSupport;

public class TestJobListener extends JobListenerSupport {
  private String name = UUID.randomUUID().toString();
  private int expectedNumOfJobFired;
  private volatile AtomicInteger currentNumOfFire = new AtomicInteger(0);

  public TestJobListener(int expectedNumOfJobFired) {
    this.expectedNumOfJobFired = expectedNumOfJobFired;
  }

  @Override
  public String getName() {
    return name;
  }

  @Override
  public void jobWasExecuted(JobExecutionContext context, JobExecutionException jobException) {
    synchronized (this) {
      if (currentNumOfFire.addAndGet(1) < expectedNumOfJobFired) {
        return;
      }

      notify();
    }
  }

  public synchronized void waitForJobToFinish(long timeoutMillis) throws InterruptedException {
    wait(timeoutMillis);
    if (currentNumOfFire.get() < expectedNumOfJobFired) {
      throw new RuntimeException(
          "Waiting for job time out. Number of times job fired: "
              + currentNumOfFire
              + " expected: "
              + expectedNumOfJobFired);
    }
  }
}
