package org.cloudfoundry.autoscaler.scheduler.conf;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import org.cloudfoundry.autoscaler.scheduler.conf.CfServerConfiguration.VerificationMode;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.ValueSource;
import org.springframework.boot.context.properties.bind.BindException;
import org.springframework.boot.context.properties.bind.Binder;
import org.springframework.mock.env.MockEnvironment;

class CfServerConfigurationTest {
  @Test
  void defaultsToEnforcement() {
    assertThat(new CfServerConfiguration().getXfccVerificationMode())
        .isEqualTo(VerificationMode.ENFORCE);
  }

  @ParameterizedTest
  @ValueSource(strings = {"enforce", "shadow", "off"})
  void bindsVerificationModeAndTrustPaths(String mode) {
    MockEnvironment environment =
        new MockEnvironment()
            .withProperty("cfserver.xfccVerificationMode", mode)
            .withProperty("cfserver.systemCertPath", "/platform/roots")
            .withProperty("cfserver.instanceCertPath", "/platform/instance.crt");
    CfServerConfiguration config =
        Binder.get(environment).bind("cfserver", CfServerConfiguration.class).get();
    assertThat(config.getXfccVerificationMode())
        .isEqualTo(VerificationMode.valueOf(mode.toUpperCase()));
    assertThat(config.getSystemCertPath()).isEqualTo("/platform/roots");
    assertThat(config.getInstanceCertPath()).isEqualTo("/platform/instance.crt");
  }

  @Test
  void rejectsUnknownModesInsteadOfDisablingVerification() {
    MockEnvironment environment =
        new MockEnvironment().withProperty("cfserver.xfccVerificationMode", "invalid");
    Binder binder = Binder.get(environment);
    assertThatThrownBy(() -> binder.bind("cfserver", CfServerConfiguration.class))
        .isInstanceOf(BindException.class);
  }
}
