package org.cloudfoundry.autoscaler.scheduler.conf;

import static org.assertj.core.api.Assertions.assertThat;

import java.security.Provider;
import java.security.Security;
import org.bouncycastle.crypto.CryptoServicesRegistrar;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;

class FipsSecurityProviderConfigTest {

  @BeforeAll
  static void setUp() {
    FipsSecurityProviderConfig.initialize();
  }

  @Test
  void bcfipsIsRegisteredAtPosition1() {
    Provider[] providers = Security.getProviders();
    assertThat(providers[0].getName()).isEqualTo("BCFIPS");
  }

  @Test
  void bcjsseIsRegisteredAtPosition2() {
    Provider[] providers = Security.getProviders();
    assertThat(providers[1].getName()).isEqualTo("BCJSSE");
  }

  @Test
  void approvedOnlyModeIsActive() {
    assertThat(CryptoServicesRegistrar.isInApprovedOnlyMode()).isTrue();
  }

  @Test
  void sunTlsProvidersAreRemoved() {
    assertThat(Security.getProvider("SunJSSE")).isNull();
    assertThat(Security.getProvider("SunEC")).isNull();
  }

  @Test
  void initializeIsIdempotent() {
    FipsSecurityProviderConfig.initialize();
    FipsSecurityProviderConfig.initialize();

    Provider[] providers = Security.getProviders();
    long bcfipsCount =
        java.util.Arrays.stream(providers).filter(p -> "BCFIPS".equals(p.getName())).count();
    assertThat(bcfipsCount).isEqualTo(1);
  }
}
