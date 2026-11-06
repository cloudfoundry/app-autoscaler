package org.cloudfoundry.autoscaler.scheduler.conf;

import static org.assertj.core.api.Assertions.assertThat;

import java.security.Security;
import java.util.concurrent.FutureTask;
import java.util.concurrent.TimeUnit;
import org.bouncycastle.crypto.CryptoServicesRegistrar;
import org.junit.jupiter.api.Test;
import org.springframework.boot.ssl.SslBundles;

/** Surefire gives this class a fresh JVM; FIPS is activated only by the VCAP startup path. */
class FipsWarStartupTest {

  @Test
  void vcapEnablesApprovedOnlyModeBeforeSslBundleCreation() throws Exception {
    assertThat(Security.getProvider("BCFIPS")).isNull();
    try (var context = FipsStartupSupport.start("{\"autoscaler\": {\"fips_mode\": true}}", true)) {
      var bundle = context.getBean(SslBundles.class).getBundle("scalingengine");
      assertThat(bundle.getManagers().getKeyManagers()).isNotEmpty();
      assertThat(bundle.createSslContext().getProvider().getName()).isEqualTo("BCJSSE");

      FutureTask<Boolean> workerMode =
          new FutureTask<>(CryptoServicesRegistrar::isInApprovedOnlyMode);
      new Thread(workerMode).start();
      assertThat(workerMode.get(10, TimeUnit.SECONDS)).isTrue();
    }
  }
}
