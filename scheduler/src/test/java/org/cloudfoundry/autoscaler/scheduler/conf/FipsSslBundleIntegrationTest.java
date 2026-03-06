package org.cloudfoundry.autoscaler.scheduler.conf;

import static org.assertj.core.api.Assertions.assertThat;

import java.security.KeyStore;
import java.security.cert.Certificate;
import javax.net.ssl.SSLContext;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;
import org.springframework.boot.ssl.SslBundle;
import org.springframework.boot.ssl.pem.PemSslStoreBundle;
import org.springframework.boot.ssl.pem.PemSslStoreDetails;

class FipsSslBundleIntegrationTest {

  @BeforeAll
  static void setUp() {
    FipsSecurityProviderConfig.initialize();
  }

  @Test
  void pemSslStoreBundleLoadsKeystoreUnderFips() throws Exception {
    PemSslStoreDetails keystoreDetails =
        PemSslStoreDetails.forCertificate("file:src/test/resources/certs/test-scheduler.crt")
            .withPrivateKey("file:src/test/resources/certs/test-scheduler.key");
    PemSslStoreDetails truststoreDetails =
        PemSslStoreDetails.forCertificate("file:src/test/resources/certs/test-ca.crt");

    PemSslStoreBundle storeBundle = new PemSslStoreBundle(keystoreDetails, truststoreDetails);

    KeyStore keyStore = storeBundle.getKeyStore();
    assertThat(keyStore).isNotNull();
    assertThat(keyStore.size()).isGreaterThan(0);

    KeyStore trustStore = storeBundle.getTrustStore();
    assertThat(trustStore).isNotNull();
    assertThat(trustStore.size()).isGreaterThan(0);

    Certificate cert = trustStore.getCertificate(trustStore.aliases().nextElement());
    assertThat(cert).isNotNull();
    assertThat(cert.getType()).isEqualTo("X.509");
  }

  @Test
  void sslContextCreationUsesBcjsse() throws Exception {
    SSLContext sslContext = SSLContext.getInstance("TLS");
    sslContext.init(null, null, null);

    assertThat(sslContext.getProvider().getName()).isEqualTo("BCJSSE");
  }
}
