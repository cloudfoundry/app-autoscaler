package org.cloudfoundry.autoscaler.scheduler.filter;

import static org.assertj.core.api.Assertions.assertThat;

import java.security.cert.CertPathBuilder;
import java.security.cert.CertificateFactory;
import org.bouncycastle.crypto.CryptoServicesRegistrar;
import org.cloudfoundry.autoscaler.scheduler.conf.FipsSecurityProviderConfig;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;

/** Surefire uses a fresh JVM so provider initialization happens before any certificate parsing. */
class FipsXfccChainVerifierTest extends XfccChainVerifierTest {
  @BeforeAll
  static void enableFips() {
    FipsSecurityProviderConfig.initialize();
  }

  @Test
  void pkixAndCertificateParsingUseApprovedOnlyBcfips() throws Exception {
    assertThat(CryptoServicesRegistrar.isInApprovedOnlyMode()).isTrue();
    assertThat(CertPathBuilder.getInstance("PKIX").getProvider().getName()).isEqualTo("BCFIPS");
    assertThat(CertificateFactory.getInstance("X.509").getProvider().getName()).isEqualTo("BCFIPS");
  }
}
