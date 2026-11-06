package org.cloudfoundry.autoscaler.scheduler.conf;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import java.nio.file.Files;
import java.nio.file.Path;
import java.security.KeyFactory;
import java.security.KeyStore;
import java.security.cert.Certificate;
import java.security.cert.CertificateFactory;
import javax.net.ssl.SSLContext;
import javax.net.ssl.X509KeyManager;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.CsvSource;
import org.junit.jupiter.params.provider.ValueSource;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.AutoConfigurations;
import org.springframework.boot.autoconfigure.ssl.SslAutoConfiguration;
import org.springframework.boot.ssl.SslBundle;
import org.springframework.boot.ssl.SslBundles;
import org.springframework.boot.ssl.pem.PemContent;
import org.springframework.boot.ssl.pem.PemSslStoreBundle;
import org.springframework.boot.ssl.pem.PemSslStoreDetails;
import org.springframework.boot.test.context.runner.ApplicationContextRunner;

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

  @ParameterizedTest
  @CsvSource({"test-scheduler.key, RSA", "test-ec.key, EC", "test-rsa-pkcs8.key, RSA"})
  void springBootParsesPrivateKeysUnderFips(String file, String algorithm) throws Exception {
    var key = PemContent.load(Path.of("src/test/resources/certs", file)).getPrivateKey();

    assertThat(key).isNotNull();
    assertThat(key.getAlgorithm()).isEqualTo(algorithm);
    assertThat(key.getClass().getName()).startsWith("org.bouncycastle.");
    assertThat(KeyFactory.getInstance(algorithm).getProvider().getName()).isEqualTo("BCFIPS");
  }

  @Test
  void springBootPreservesCertificateChainUnderFips() throws Exception {
    String leaf = Files.readString(Path.of("src/test/resources/certs/test-scheduler.crt"));
    String ca = Files.readString(Path.of("src/test/resources/certs/test-ca.crt"));
    var certificates = PemContent.of(leaf + "\n" + ca).getCertificates();

    assertThat(certificates).hasSize(2);
    assertThat(certificates.get(0).getSubjectX500Principal().getName()).contains("test-scheduler");
    assertThat(certificates.get(1).getSubjectX500Principal().getName()).contains("testCA");
    assertThat(CertificateFactory.getInstance("X.509").getProvider().getName()).isEqualTo("BCFIPS");

    var store =
        new PemSslStoreBundle(
            PemSslStoreDetails.forCertificate(leaf + "\n" + ca)
                .withPrivateKey("file:src/test/resources/certs/test-scheduler.key"),
            null);
    assertThat(store.getKeyStore().getCertificateChain("ssl"))
        .containsExactlyElementsOf(certificates);
    var bundle = SslBundle.of(store);
    assertThat(bundle.getManagers().getKeyManagers()).isNotEmpty();
    assertThat(bundle.createSslContext().getProvider().getName()).isEqualTo("BCJSSE");
  }

  @ParameterizedTest
  @ValueSource(booleans = {false, true})
  void cfInstanceIdentityIsAvailableThroughSpringBootBundle(
      boolean includeCaCertificate, @TempDir Path tempDir) throws Exception {
    String leaf = Files.readString(Path.of("src/test/resources/certs/test-scheduler.crt"));
    String ca = Files.readString(Path.of("src/test/resources/certs/test-ca.crt"));
    Path chainFile = Files.writeString(tempDir.resolve("instance.crt"), leaf + "\n" + ca);

    var runner =
        new ApplicationContextRunner()
            .withConfiguration(AutoConfigurations.of(SslAutoConfiguration.class))
            .withPropertyValues(
                "VCAP_SERVICES={}",
                "CF_INSTANCE_CERT=" + chainFile,
                "CF_INSTANCE_KEY=src/test/resources/certs/test-scheduler.key")
            .withInitializer(
                context ->
                    new CloudFoundryConfigurationProcessor()
                        .postProcessEnvironment(context.getEnvironment(), new SpringApplication()));
    if (includeCaCertificate) {
      runner = runner.withPropertyValues("CF_INSTANCE_CA_CERT=src/test/resources/certs/test-ca.crt");
    }
    runner.run(
            context -> {
              assertThat(context).hasNotFailed();
              var bundle = context.getBean(SslBundles.class).getBundle("scalingengine");
              var keyManager = (X509KeyManager) bundle.getManagers().getKeyManagers()[0];
              String alias = keyManager.chooseClientAlias(new String[] {"RSA"}, null, null);

              assertThat(alias).isNotNull();
              assertThat(keyManager.getPrivateKey(alias)).isNotNull();
              assertThat(keyManager.getCertificateChain(alias))
                  .containsExactlyElementsOf(PemContent.of(leaf + "\n" + ca).getCertificates());
              if (!includeCaCertificate) {
                assertThat(bundle.getStores().getTrustStore()).isNull();
              }
              assertThat(bundle.createSslContext().getProvider().getName()).isEqualTo("BCJSSE");
            });
  }

  @Test
  void invalidPemIdentityFailsInsteadOfDisablingClientAuthentication() {
    assertThatThrownBy(
            () ->
                new PemSslStoreBundle(
                        PemSslStoreDetails.forCertificate(
                                "file:src/test/resources/certs/test-scheduler.crt")
                            .withPrivateKey(
                                "-----BEGIN PRIVATE KEY-----\ngarbage\n-----END PRIVATE KEY-----"),
                        null)
                    .getKeyStore())
        .isInstanceOf(IllegalStateException.class);
  }
}
