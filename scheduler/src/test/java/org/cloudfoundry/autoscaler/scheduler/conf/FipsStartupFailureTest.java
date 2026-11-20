package org.cloudfoundry.autoscaler.scheduler.conf;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import org.bouncycastle.crypto.CryptoServicesRegistrar;
import org.junit.jupiter.api.Test;

/** Isolated JVM: deliberately initialize BC too early to verify that startup fails closed. */
class FipsStartupFailureTest {

  @Test
  void unapprovedProviderPreventsApplicationStartup() {
    assertThat(CryptoServicesRegistrar.isInApprovedOnlyMode()).isFalse();

    assertThatThrownBy(
            () -> FipsStartupSupport.start("{\"autoscaler\": {\"fips_mode\": true}}", true))
        .isInstanceOf(IllegalStateException.class)
        .hasMessageContaining("Failed to activate FIPS approved-only mode");
    assertThat(FipsSecurityProviderConfig.isInitialized()).isFalse();
  }
}
