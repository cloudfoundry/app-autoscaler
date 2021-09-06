package org.cloudfoundry.autoscaler.scheduler.conf;

import static org.junit.jupiter.api.Assertions.assertThrows;

import org.junit.jupiter.api.Test;

class MetricsConfigurationTest {
  @Test
  void givenBasicAuthEnableAndUsernameOrPasswordIsNull() {
    assertThrows(
        IllegalStateException.class, () -> new MetricsConfiguration(null, null, 8081, true).init());
  }

  @Test
  void givenBasicAuthEnableAndUsernameOrPasswordIsEmpty() {
    assertThrows(
        IllegalStateException.class, () -> new MetricsConfiguration("", "", 8081, true).init());
  }
}
