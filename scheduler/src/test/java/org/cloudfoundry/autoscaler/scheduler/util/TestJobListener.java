package org.cloudfoundry.autoscaler.scheduler.util;

import java.util.UUID;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;
import org.quartz.JobExecutionContext;
import org.quartz.JobExecutionException;
import org.quartz.listeners.JobListenerSupport;

public class TestJobListener extends JobListenerSupport {
  private final String name = UUID.randomUUID().toString();
  private final CountDownLatch latch;
  private final int expectedNumOfJobFired;
  private final AtomicInteger currentNumOfFire = new AtomicInteger(0);

  public TestJobListener(int expectedNumOfJobFired) {
    this.expectedNumOfJobFired = expectedNumOfJobFired;
    this.latch = new CountDownLatch(expectedNumOfJobFired);
  }

  @Override
  public String getName() {
    return name;
  }

  @Override
  public void jobWasExecuted(JobExecutionContext context, JobExecutionException jobException) {
    currentNumOfFire.incrementAndGet();
    latch.countDown();
  }

  public void waitForJobToFinish(long timeoutMillis) throws InterruptedException {
    if (!latch.await(timeoutMillis, TimeUnit.MILLISECONDS)) {
      throw new RuntimeException(
          "Waiting for job time out. Number of times job fired: "
              + currentNumOfFire.get()
              + " expected: "
              + expectedNumOfJobFired);
    }
  }
}
