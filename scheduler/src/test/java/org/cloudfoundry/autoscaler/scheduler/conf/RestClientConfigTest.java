package org.cloudfoundry.autoscaler.scheduler.conf;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import com.sun.net.httpserver.HttpsConfigurator;
import com.sun.net.httpserver.HttpsExchange;
import com.sun.net.httpserver.HttpsParameters;
import com.sun.net.httpserver.HttpsServer;
import java.io.IOException;
import java.net.InetAddress;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.security.cert.Certificate;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.atomic.AtomicReference;
import javax.net.ssl.SSLContext;
import org.apache.hc.client5.http.classic.methods.HttpGet;
import org.apache.hc.client5.http.impl.classic.CloseableHttpClient;
import org.apache.hc.core5.http.HttpHost;
import org.apache.hc.core5.http.io.entity.EntityUtils;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.CsvSource;
import org.springframework.boot.ssl.DefaultSslBundleRegistry;
import org.springframework.boot.ssl.SslBundle;
import org.springframework.boot.ssl.SslBundleKey;
import org.springframework.boot.ssl.SslStoreBundle;
import org.springframework.boot.ssl.pem.PemContent;
import org.springframework.boot.ssl.pem.PemSslStoreBundle;
import org.springframework.boot.ssl.pem.PemSslStoreDetails;

class RestClientConfigTest {

  private final Map<String, String> originalProperties = new HashMap<>();
  private final AtomicReference<Certificate[]> peerCertificates = new AtomicReference<>();
  private HttpsServer server;

  @BeforeAll
  static void enableFips() {
    FipsSecurityProviderConfig.initialize();
  }

  @BeforeEach
  void saveTrustStoreProperties() {
    for (String name : List.of("trustStore", "trustStorePassword", "trustStoreType")) {
      String key = "javax.net.ssl." + name;
      originalProperties.put(key, System.getProperty(key));
    }
  }

  @AfterEach
  void cleanUp() {
    if (server != null) {
      server.stop(0);
    }
    originalProperties.forEach(
        (key, value) -> {
          if (value == null) {
            System.clearProperty(key);
          } else {
            System.setProperty(key, value);
          }
        });
  }

  @ParameterizedTest
  @CsvSource({
    "SYSTEM, TLSv1.2", "CF, TLSv1.2", "CUSTOM, TLSv1.2",
    "SYSTEM, TLSv1.3", "CF, TLSv1.3", "CUSTOM, TLSv1.3"
  })
  void trustsEachSourceAndSendsClientIdentity(String trustSource, String protocol)
      throws Exception {
    startServer();
    if (trustSource.equals("SYSTEM")) {
      System.setProperty("javax.net.ssl.trustStore", "src/test/resources/certs/test.truststore");
      System.setProperty("javax.net.ssl.trustStorePassword", "123456");
      System.setProperty("javax.net.ssl.trustStoreType", "PKCS12");
    }
    SslBundle bundle = clientBundle(trustSource.equals("CUSTOM"));
    String cfCaCerts = trustSource.equals("CF") ? "src/test/resources/certs/test-ca.crt" : "";
    RestClientConfig config = config(bundle, protocol, cfCaCerts);

    assertThat(get(config, protocol)).isEqualTo("ok");
    assertThat(peerCertificates.get())
        .containsExactlyElementsOf(PemContent.of(clientChain()).getCertificates());
  }

  @Test
  void rejectsServerOutsideComposedTrust() throws Exception {
    startServer();
    RestClientConfig config = config(clientBundle(false), "TLSv1.2", "");

    assertThatThrownBy(() -> getAtHost(config, "localhost"))
        .isInstanceOf(IOException.class)
        .hasStackTraceContaining("CertificateException");
    assertThat(peerCertificates.get()).isNull();
  }

  @Test
  void rejectsWrongHostnameEvenWithTrustedCa() throws Exception {
    startServer();
    RestClientConfig config = config(clientBundle(true), "TLSv1.2", "");

    assertThatThrownBy(() -> getAtHost(config, "wrong-host.invalid"))
        .isInstanceOf(IOException.class)
        .hasRootCauseInstanceOf(java.security.cert.CertificateException.class)
        .rootCause()
        .hasMessageContaining("wrong-host.invalid");
    assertThat(peerCertificates.get()).isNull();
  }

  @Test
  void invalidConfiguredKeyPasswordFailsStartup() throws Exception {
    var stores = clientBundle(true).getStores();
    var bundle =
        SslBundle.of(
            SslStoreBundle.of(stores.getKeyStore(), "wrong-password", stores.getTrustStore()));

    assertThatThrownBy(() -> config(bundle, "TLSv1.2", ""))
        .isInstanceOf(IllegalStateException.class)
        .hasMessageContaining("key manager");
  }

  @Test
  void unreadableConfiguredTrustStoreFailsStartup() {
    SslStoreBundle broken =
        new SslStoreBundle() {
          @Override
          public java.security.KeyStore getKeyStore() {
            return null;
          }

          @Override
          public String getKeyStorePassword() {
            return null;
          }

          @Override
          public java.security.KeyStore getTrustStore() {
            throw new IllegalStateException("unreadable truststore");
          }
        };
    assertThatThrownBy(() -> config(SslBundle.of(broken), "TLSv1.2", ""))
        .isInstanceOf(IllegalStateException.class)
        .hasMessageContaining("unreadable truststore");
  }

  private void startServer() throws Exception {
    var stores =
        new PemSslStoreBundle(
            PemSslStoreDetails.forCertificate(
                    "file:src/test/resources/certs/test-scalingengine.crt")
                .withPrivateKey("file:src/test/resources/certs/test-scalingengine.key"),
            PemSslStoreDetails.forCertificate("file:src/test/resources/certs/test-ca.crt"));
    SSLContext context = SslBundle.of(stores).createSslContext();
    assertThat(context.getProvider().getName()).isEqualTo("BCJSSE");

    server = HttpsServer.create(new InetSocketAddress(InetAddress.getByName("localhost"), 0), 0);
    server.setHttpsConfigurator(
        new HttpsConfigurator(context) {
          @Override
          public void configure(HttpsParameters parameters) {
            var sslParameters = context.getDefaultSSLParameters();
            sslParameters.setNeedClientAuth(true);
            parameters.setSSLParameters(sslParameters);
          }
        });
    server.createContext(
        "/",
        exchange -> {
          try (exchange) {
            peerCertificates.set(((HttpsExchange) exchange).getSSLSession().getPeerCertificates());
            byte[] response = "ok".getBytes(StandardCharsets.UTF_8);
            exchange.sendResponseHeaders(200, response.length);
            exchange.getResponseBody().write(response);
          }
        });
    server.start();
  }

  private String get(RestClientConfig config, String protocol) throws Exception {
    try (var client = (CloseableHttpClient) config.httpClient(protocol, 5)) {
      var operations = config.restOperations(config.clientHttpRequestFactory(client));
      return operations.getForObject(
          "https://localhost:" + server.getAddress().getPort(), String.class);
    }
  }

  private static RestClientConfig config(SslBundle bundle, String protocol, String cfCaCerts) {
    var bundles = new DefaultSslBundleRegistry();
    bundles.registerBundle("scalingengine", bundle);
    return new RestClientConfig(bundles, protocol, cfCaCerts);
  }

  private String getAtHost(RestClientConfig config, String hostname) throws Exception {
    // Pin the loopback address so an IPv4/IPv6 retry cannot mask a TLS validation failure.
    var target =
        new HttpHost(
            "https", server.getAddress().getAddress(), hostname, server.getAddress().getPort());
    try (var client = (CloseableHttpClient) config.httpClient("TLSv1.2", 5)) {
      return client.execute(
          target, new HttpGet("/"), response -> EntityUtils.toString(response.getEntity()));
    }
  }

  private static SslBundle clientBundle(boolean customTrust) throws Exception {
    var stores =
        new PemSslStoreBundle(
            PemSslStoreDetails.forCertificate(clientChain())
                .withPrivateKey("file:src/test/resources/certs/test-scheduler.key")
                .withAlias("cf-instance-identity"),
            customTrust
                ? PemSslStoreDetails.forCertificate("file:src/test/resources/certs/test-ca.crt")
                : null);
    return SslBundle.of(stores, SslBundleKey.of(null, "cf-instance-identity"));
  }

  private static String clientChain() throws Exception {
    return Files.readString(Path.of("src/test/resources/certs/test-scheduler.crt"))
        + Files.readString(Path.of("src/test/resources/certs/test-ca.crt"));
  }
}
