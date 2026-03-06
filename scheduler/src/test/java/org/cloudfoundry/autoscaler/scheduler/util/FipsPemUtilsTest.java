package org.cloudfoundry.autoscaler.scheduler.util;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import java.nio.file.Files;
import java.nio.file.Path;
import java.security.PrivateKey;
import java.security.cert.CertificateException;
import java.security.cert.X509Certificate;
import org.cloudfoundry.autoscaler.scheduler.conf.FipsSecurityProviderConfig;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;

class FipsPemUtilsTest {

  @BeforeAll
  static void setUp() {
    FipsSecurityProviderConfig.initialize();
  }

  @Test
  void parseCertificateWithValidPem() throws Exception {
    String pem = Files.readString(Path.of("src/test/resources/certs/test-scheduler.crt"));
    X509Certificate cert = FipsPemUtils.parseCertificate(pem);

    assertThat(cert).isNotNull();
    assertThat(cert.getSubjectX500Principal().getName()).contains("test-scheduler");
  }

  @Test
  void parseCertificateWithCaCert() throws Exception {
    String pem = Files.readString(Path.of("src/test/resources/certs/test-ca.crt"));
    X509Certificate cert = FipsPemUtils.parseCertificate(pem);

    assertThat(cert).isNotNull();
    assertThat(cert.getSubjectX500Principal().getName()).contains("testCA");
  }

  @Test
  void parseCertificateWithNullInputThrows() {
    assertThatThrownBy(() -> FipsPemUtils.parseCertificate(null))
        .isInstanceOf(CertificateException.class)
        .hasMessageContaining("null or empty");
  }

  @Test
  void parseCertificateWithEmptyInputThrows() {
    assertThatThrownBy(() -> FipsPemUtils.parseCertificate(""))
        .isInstanceOf(CertificateException.class)
        .hasMessageContaining("null or empty");
  }

  @Test
  void parseCertificateWithInvalidPemThrows() {
    assertThatThrownBy(() -> FipsPemUtils.parseCertificate("not a cert"))
        .isInstanceOf(IllegalArgumentException.class);
  }

  @Test
  void parseRsaPrivateKey() throws Exception {
    String pem = Files.readString(Path.of("src/test/resources/certs/test-scheduler.key"));
    PrivateKey key = FipsPemUtils.parsePrivateKey(pem);

    assertThat(key).isNotNull();
    assertThat(key.getAlgorithm()).isEqualTo("RSA");
  }

  @Test
  void parsePrivateKeyWithNullInputThrows() {
    assertThatThrownBy(() -> FipsPemUtils.parsePrivateKey(null))
        .isInstanceOf(IllegalArgumentException.class)
        .hasMessageContaining("null or empty");
  }

  @Test
  void parsePrivateKeyWithEmptyInputThrows() {
    assertThatThrownBy(() -> FipsPemUtils.parsePrivateKey(""))
        .isInstanceOf(IllegalArgumentException.class)
        .hasMessageContaining("null or empty");
  }
}
