package org.cloudfoundry.autoscaler.scheduler.util;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import java.nio.file.Files;
import java.nio.file.Path;
import java.security.KeyStore;
import java.security.cert.Certificate;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import org.springframework.boot.ssl.pem.PemContent;

class CfTrustStoreTest {

  @TempDir Path tempDir;

  private final Map<String, String> originalProperties = new HashMap<>();

  @BeforeEach
  void saveTrustStoreProperties() {
    for (String name : List.of("trustStore", "trustStorePassword", "trustStoreType")) {
      String key = "javax.net.ssl." + name;
      originalProperties.put(key, System.getProperty(key));
    }
  }

  @AfterEach
  void restoreTrustStoreProperties() {
    originalProperties.forEach(
        (key, value) -> {
          if (value == null) {
            System.clearProperty(key);
          } else {
            System.setProperty(key, value);
          }
        });
  }

  @Test
  void composesAllSourcesWithoutDuplicatesOrMutatingConfiguredStore() throws Exception {
    Certificate system = certificate("test-ca.crt");
    Certificate cf = certificate("test-scheduler.crt");
    Certificate custom = certificate("test-scalingengine.crt");

    KeyStore defaultStore = emptyStore();
    defaultStore.setCertificateEntry("system", system);
    Path defaultPath = tempDir.resolve("custom-default.jks");
    try (var output = Files.newOutputStream(defaultPath)) {
      defaultStore.store(output, "different-password".toCharArray());
    }
    System.setProperty("javax.net.ssl.trustStore", defaultPath.toString());
    System.setProperty("javax.net.ssl.trustStorePassword", "different-password");
    System.setProperty("javax.net.ssl.trustStoreType", "JKS");

    Path cfPath = tempDir.resolve("cf-ca-certs.pem");
    Files.writeString(cfPath, pem("test-ca.crt") + pem("test-scheduler.crt"));
    KeyStore configured = emptyStore();
    configured.setCertificateEntry("custom", custom);
    configured.setCertificateEntry("duplicate", cf);

    KeyStore combined = CfTrustStore.compose(configured, cfPath);

    assertThat(combined.size()).isEqualTo(3);
    assertThat(combined.getCertificateAlias(system)).isNotNull();
    assertThat(combined.getCertificateAlias(cf)).isNotNull();
    assertThat(combined.getCertificateAlias(custom)).isNotNull();
    for (String alias : Collections.list(combined.aliases())) {
      assertThat(combined.isCertificateEntry(alias)).isTrue();
    }
    assertThat(configured.size()).isEqualTo(2);
    assertThat(configured.getCertificate("custom")).isEqualTo(custom);
    assertThat(configured.getCertificate("duplicate")).isEqualTo(cf);
  }

  @Test
  void supportsDefaultCfLocationAndNoCustomStore() throws Exception {
    assertThat(CfTrustStore.compose(null, null).size()).isGreaterThan(0);
  }

  @Test
  void missingExplicitCfFileFails() {
    assertThatThrownBy(() -> CfTrustStore.compose(null, tempDir.resolve("missing.pem")))
        .isInstanceOf(IllegalStateException.class)
        .hasRootCauseInstanceOf(java.nio.file.NoSuchFileException.class);
  }

  @Test
  void malformedCfCertificatesFail() throws Exception {
    Path invalid = Files.writeString(tempDir.resolve("invalid.pem"), "not a certificate");
    assertThatThrownBy(() -> CfTrustStore.compose(null, invalid))
        .isInstanceOf(IllegalStateException.class)
        .hasMessageContaining("outbound HTTPS truststore");
  }

  private static KeyStore emptyStore() throws Exception {
    KeyStore store = KeyStore.getInstance("JKS");
    store.load(null, null);
    return store;
  }

  private static String pem(String file) throws Exception {
    return Files.readString(Path.of("src/test/resources/certs", file));
  }

  private static Certificate certificate(String file) throws Exception {
    return PemContent.of(pem(file)).getCertificates().get(0);
  }
}
