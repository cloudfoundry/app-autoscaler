package org.cloudfoundry.autoscaler.scheduler.conf;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import java.security.Security;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.ValueSource;
import org.springframework.boot.ssl.SslBundles;

class FipsDisabledStartupTest {

  @ParameterizedTest
  @ValueSource(strings = {"{}", "{\"autoscaler\": {\"fips_mode\": false}}"})
  void missingOrFalseFlagKeepsDefaultProviders(String credentials) {
    try (var context = FipsStartupSupport.start(credentials, false)) {
      assertThat(Security.getProvider("BCFIPS")).isNull();
      var bundle = context.getBean(SslBundles.class).getBundle("scalingengine");
      assertThat(bundle.createSslContext().getProvider().getName()).isEqualTo("SunJSSE");
    }
  }

  @Test
  void invalidFlagFailsStartup() {
    assertThatThrownBy(
            () -> FipsStartupSupport.start("{\"autoscaler\": {\"fips_mode\": \"invalid\"}}", false))
        .isInstanceOf(org.springframework.boot.context.properties.bind.BindException.class);
    assertThat(Security.getProvider("BCFIPS")).isNull();
  }
}
