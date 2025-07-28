package org.cloudfoundry.autoscaler.scheduler.util;

import java.util.TimeZone;
import org.junit.rules.TestRule;
import org.junit.runner.Description;
import org.junit.runners.model.Statement;

/** Rule for running the test classes with different Time zones */
public class TimeZoneTestRule implements TestRule {
  private String[] zoneIds;

  public TimeZoneTestRule(String[] zoneIds) {
    this.zoneIds = zoneIds;
  }

  @Override
  public Statement apply(final Statement base, Description description) {
    return new Statement() {

      @Override
      public void evaluate() throws Throwable {
        TimeZone defaultTimeZone = TimeZone.getDefault();
        try {
          for (String zoneId : zoneIds) {
            TimeZone.setDefault(TimeZone.getTimeZone(zoneId));
            base.evaluate();
          }

        } finally {
          TimeZone.setDefault(defaultTimeZone);
        }
      }
    };
  }
}
