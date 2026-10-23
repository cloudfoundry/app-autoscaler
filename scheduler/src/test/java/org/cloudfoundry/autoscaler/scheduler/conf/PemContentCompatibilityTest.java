package org.cloudfoundry.autoscaler.scheduler.conf;

import static org.assertj.core.api.Assertions.assertThat;

import java.nio.file.Files;
import java.nio.file.Path;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.CsvSource;
import org.springframework.boot.ssl.pem.PemContent;

/** Verifies PEM loading without registering the FIPS providers in this test JVM. */
class PemContentCompatibilityTest {

  @ParameterizedTest
  @CsvSource({"test-scheduler.key, RSA", "test-ec.key, EC", "test-rsa-pkcs8.key, RSA"})
  void parsesPrivateKeysWithDefaultProviders(String file, String algorithm) throws Exception {
    var key = PemContent.load(Path.of("src/test/resources/certs", file)).getPrivateKey();

    assertThat(key).isNotNull();
    assertThat(key.getAlgorithm()).isEqualTo(algorithm);
  }

  @Test
  void preservesCertificateChainWithDefaultProviders() throws Exception {
    String leaf = Files.readString(Path.of("src/test/resources/certs/test-scheduler.crt"));
    String ca = Files.readString(Path.of("src/test/resources/certs/test-ca.crt"));
    var certificates = PemContent.of(leaf + "\n" + ca).getCertificates();

    assertThat(certificates).hasSize(2);
    assertThat(certificates.get(0).getSubjectX500Principal().getName()).contains("test-scheduler");
    assertThat(certificates.get(1).getSubjectX500Principal().getName()).contains("testCA");
  }
}
