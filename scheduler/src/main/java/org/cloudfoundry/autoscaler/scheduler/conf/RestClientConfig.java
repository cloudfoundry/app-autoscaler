package org.cloudfoundry.autoscaler.scheduler.conf;

import java.net.Socket;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.security.KeyStore;
import java.security.Principal;
import java.security.PrivateKey;
import java.security.cert.X509Certificate;
import javax.net.ssl.SSLContext;
import javax.net.ssl.TrustManagerFactory;
import javax.net.ssl.X509KeyManager;
import javax.net.ssl.X509TrustManager;
import org.apache.hc.client5.http.classic.HttpClient;
import org.apache.hc.client5.http.config.ConnectionConfig;
import org.apache.hc.client5.http.config.RequestConfig;
import org.apache.hc.client5.http.impl.classic.HttpClientBuilder;
import org.apache.hc.client5.http.impl.io.PoolingHttpClientConnectionManagerBuilder;
import org.apache.hc.client5.http.io.HttpClientConnectionManager;
import org.apache.hc.client5.http.ssl.HttpsSupport;
import org.apache.hc.client5.http.ssl.SSLConnectionSocketFactory;
import org.apache.hc.core5.util.Timeout;
import org.cloudfoundry.autoscaler.scheduler.util.FipsPemUtils;
import org.cloudfoundry.autoscaler.scheduler.util.FipsSslContextBuilder;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.ssl.SslBundle;
import org.springframework.boot.ssl.SslBundles;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.http.client.ClientHttpRequestFactory;
import org.springframework.http.client.HttpComponentsClientHttpRequestFactory;
import org.springframework.web.client.RestOperations;
import org.springframework.web.client.RestTemplate;

@Configuration
public class RestClientConfig {
  private static final Logger logger = LoggerFactory.getLogger(RestClientConfig.class);
  private final SSLContext sslContext;

  @Autowired
  public RestClientConfig(
      SslBundles sslBundles, @Value("${client.ssl.protocol}") String protocol) {
    SslBundle sslBundle = sslBundles.getBundle("scalingengine");

    // Load custom CA certificates from SSL bundle if configured (for test environments)
    X509TrustManager customTrustManager = loadCustomTrustManager(sslBundle);

    // Load client certificate KeyManagers for mutual TLS authentication.
    // Try SSL bundle first, then fall back to reading CF_INSTANCE_CERT/KEY directly.
    javax.net.ssl.KeyManager[] keyManagers = loadKeyManagers(sslBundle);
    if (keyManagers == null) {
      keyManagers = loadCfInstanceKeyManagers();
    }
    keyManagers = wrapWithDebugLogging(keyManagers);

    // In FIPS mode, Bouncy Castle TrustManager doesn't automatically fall back
    // to system trust, so we explicitly merge system CAs with custom CAs.
    if (customTrustManager != null) {
      this.sslContext =
          FipsSslContextBuilder.buildWithSystemAndCustomTrust(
              keyManagers, customTrustManager, protocol);
      logger.info(
          "RestClientConfig initialized with FIPS-compliant SSLContext using protocol: {} "
              + "with system CA trust anchors and {} custom CA certificate(s){}",
          protocol,
          customTrustManager.getAcceptedIssuers().length,
          keyManagers != null ? " and client certificate" : "");
    } else {
      this.sslContext =
          FipsSslContextBuilder.buildWithSystemTrust(keyManagers, protocol);
      logger.info(
          "RestClientConfig initialized with FIPS-compliant SSLContext using protocol: {} "
              + "with system CA trust anchors only{}",
          protocol,
          keyManagers != null ? " and client certificate" : "");
    }
  }

  /**
   * Loads KeyManagers from SSL bundle for client certificate authentication (mutual TLS).
   *
   * @param sslBundle SSL bundle configuration
   * @return KeyManager array or null if not configured
   */
  private javax.net.ssl.KeyManager[] loadKeyManagers(SslBundle sslBundle) {
    try {
      KeyStore keyStore = sslBundle.getStores().getKeyStore();

      if (keyStore == null) {
        logger.info("No keystore configured in SSL bundle");
        return null;
      }

      if (keyStore.size() == 0) {
        logger.info("Keystore from SSL bundle is empty");
        return null;
      }

      logger.info("Loaded keystore with {} entry/entries from SSL bundle", keyStore.size());

      javax.net.ssl.KeyManagerFactory kmf =
          javax.net.ssl.KeyManagerFactory.getInstance(
              javax.net.ssl.KeyManagerFactory.getDefaultAlgorithm());
      kmf.init(keyStore, null);

      logger.info(
          "KeyManagerFactory initialized successfully with {} KeyManager(s)",
          kmf.getKeyManagers().length);
      return kmf.getKeyManagers();

    } catch (Exception e) {
      logger.info("Could not load keystore from SSL bundle: {}", e.getMessage());
      return null;
    }
  }

  /**
   * Loads KeyManagers directly from CF_INSTANCE_CERT and CF_INSTANCE_KEY file paths. This is the
   * fallback when the Spring SSL bundle doesn't contain the client certificate (e.g., because the
   * EnvironmentPostProcessor couldn't inject them, or PEM parsing failed under FIPS).
   *
   * <p>CF_INSTANCE_CERT and CF_INSTANCE_KEY are file paths set by Diego to give each container a
   * unique cryptographic identity for mutual TLS.
   *
   * @return KeyManager array or null if CF instance identity is not available
   */
  private javax.net.ssl.KeyManager[] loadCfInstanceKeyManagers() {
    String certPath = System.getenv("CF_INSTANCE_CERT");
    String keyPath = System.getenv("CF_INSTANCE_KEY");

    if (certPath == null || keyPath == null) {
      logger.info(
          "CF_INSTANCE_CERT/CF_INSTANCE_KEY not set - no CF instance identity available");
      return null;
    }

    try {
      Path certFile = Paths.get(certPath);
      Path keyFile = Paths.get(keyPath);

      if (!Files.exists(certFile) || !Files.exists(keyFile)) {
        logger.warn(
            "CF instance cert/key files do not exist: cert={} (exists={}), key={} (exists={})",
            certPath,
            Files.exists(certFile),
            keyPath,
            Files.exists(keyFile));
        return null;
      }

      String certPem = Files.readString(certFile);
      String keyPem = Files.readString(keyFile);

      logger.info(
          "Loading CF instance identity from {} and {}", certPath, keyPath);

      X509Certificate cert = FipsPemUtils.parseCertificate(certPem);
      PrivateKey privateKey = FipsPemUtils.parsePrivateKey(keyPem);

      KeyStore keyStore = KeyStore.getInstance(KeyStore.getDefaultType());
      keyStore.load(null, null);
      keyStore.setKeyEntry(
          "cf-instance-identity",
          privateKey,
          new char[0],
          new X509Certificate[] {cert});

      javax.net.ssl.KeyManagerFactory kmf =
          javax.net.ssl.KeyManagerFactory.getInstance(
              javax.net.ssl.KeyManagerFactory.getDefaultAlgorithm());
      kmf.init(keyStore, new char[0]);

      logger.info(
          "Loaded CF instance identity certificate (subject: {}, issuer: {})",
          cert.getSubjectX500Principal().getName(),
          cert.getIssuerX500Principal().getName());

      return kmf.getKeyManagers();

    } catch (Exception e) {
      logger.error("Failed to load CF instance identity certificates", e);
      return null;
    }
  }

  /**
   * Loads custom TrustManager from SSL bundle if truststore certificates are configured.
   *
   * @param sslBundle SSL bundle configuration
   * @return X509TrustManager with custom CA certificates, or null if not configured
   */
  private X509TrustManager loadCustomTrustManager(SslBundle sslBundle) {
    try {
      KeyStore trustStore = sslBundle.getStores().getTrustStore();

      if (trustStore == null) {
        logger.info("No truststore configured in SSL bundle");
        return null;
      }

      // Check if the truststore actually has certificates
      if (trustStore.size() == 0) {
        logger.info("Truststore from SSL bundle is empty");
        return null;
      }

      logger.info("Loaded truststore with {} certificate(s) from SSL bundle", trustStore.size());

      // Create TrustManager from the truststore
      TrustManagerFactory tmf =
          TrustManagerFactory.getInstance(TrustManagerFactory.getDefaultAlgorithm());
      tmf.init(trustStore);

      for (javax.net.ssl.TrustManager tm : tmf.getTrustManagers()) {
        if (tm instanceof X509TrustManager) {
          return (X509TrustManager) tm;
        }
      }

      logger.warn("No X509TrustManager found in TrustManagerFactory");
      return null;

    } catch (Exception e) {
      logger.warn(
          "Could not load custom truststore from SSL bundle, will use system CAs only: {}",
          e.getMessage());
      return null;
    }
  }

  /**
   * Wraps each X509KeyManager in a logging decorator so that every TLS handshake with the scaling
   * engine emits an INFO log showing which certificate alias (and its subject DN) was chosen.
   * Returns the original array unchanged if it is null or contains no X509KeyManager instances.
   */
  private javax.net.ssl.KeyManager[] wrapWithDebugLogging(
      javax.net.ssl.KeyManager[] keyManagers) {
    if (keyManagers == null) {
      return null;
    }
    javax.net.ssl.KeyManager[] wrapped = new javax.net.ssl.KeyManager[keyManagers.length];
    for (int i = 0; i < keyManagers.length; i++) {
      if (keyManagers[i] instanceof X509KeyManager) {
        wrapped[i] = new DebuggingX509KeyManager((X509KeyManager) keyManagers[i]);
      } else {
        wrapped[i] = keyManagers[i];
      }
    }
    return wrapped;
  }

  /**
   * X509KeyManager decorator that logs every alias selection and certificate chain lookup at INFO
   * level. This makes it possible to prove (or disprove) that a client certificate is presented
   * during TLS handshakes with the scaling engine.
   */
  private static class DebuggingX509KeyManager implements X509KeyManager {
    private static final Logger debugLogger =
        LoggerFactory.getLogger(DebuggingX509KeyManager.class);
    private final X509KeyManager delegate;

    DebuggingX509KeyManager(X509KeyManager delegate) {
      this.delegate = delegate;
    }

    @Override
    public String chooseClientAlias(String[] keyType, Principal[] issuers, Socket socket) {
      String alias = delegate.chooseClientAlias(keyType, issuers, socket);
      if (alias != null) {
        X509Certificate[] chain = delegate.getCertificateChain(alias);
        String subject =
            (chain != null && chain.length > 0)
                ? chain[0].getSubjectX500Principal().getName()
                : "<no chain>";
        debugLogger.info(
            "[mTLS] Chose client alias '{}' with subject '{}' for key types {} to {}",
            alias,
            subject,
            java.util.Arrays.toString(keyType),
            socket != null ? socket.getInetAddress() : "unknown");
      } else {
        debugLogger.info(
            "[mTLS] No client alias available for key types {} – no client certificate will be sent",
            java.util.Arrays.toString(keyType));
      }
      return alias;
    }

    @Override
    public String chooseServerAlias(String keyType, Principal[] issuers, Socket socket) {
      return delegate.chooseServerAlias(keyType, issuers, socket);
    }

    @Override
    public X509Certificate[] getCertificateChain(String alias) {
      return delegate.getCertificateChain(alias);
    }

    @Override
    public String[] getClientAliases(String keyType, Principal[] issuers) {
      return delegate.getClientAliases(keyType, issuers);
    }

    @Override
    public PrivateKey getPrivateKey(String alias) {
      return delegate.getPrivateKey(alias);
    }

    @Override
    public String[] getServerAliases(String keyType, Principal[] issuers) {
      return delegate.getServerAliases(keyType, issuers);
    }
  }

  @Bean
  public RestOperations restOperations(ClientHttpRequestFactory clientHttpRequestFactory)
      throws Exception {
    return new RestTemplate(clientHttpRequestFactory);
  }

  @Bean
  public ClientHttpRequestFactory clientHttpRequestFactory(HttpClient httpClient) {
    return new HttpComponentsClientHttpRequestFactory(httpClient);
  }

  @Bean
  public HttpClient httpClient(
      @Value("${client.ssl.protocol}") String protocol,
      @Value("${client.httpClientTimeout}") Integer httpClientTimeout)
      throws Exception {

    HttpClientBuilder builder = HttpClientBuilder.create();
    SSLConnectionSocketFactory sslsf =
        new SSLConnectionSocketFactory(
            this.sslContext,
            new String[] {protocol},
            null,
            HttpsSupport.getDefaultHostnameVerifier());

    var connectionConfig =
        ConnectionConfig.custom().setConnectTimeout(Timeout.ofSeconds(httpClientTimeout)).build();
    HttpClientConnectionManager ccm =
        PoolingHttpClientConnectionManagerBuilder.create()
            .setSSLSocketFactory(sslsf)
            .setDefaultConnectionConfig(connectionConfig)
            .build();
    builder.setConnectionManager(ccm);

    RequestConfig requestConfig =
        RequestConfig.custom()
            .setConnectionRequestTimeout(Timeout.ofSeconds(httpClientTimeout))
            .setResponseTimeout(Timeout.ofSeconds(httpClientTimeout))
            .build();
    builder.setDefaultRequestConfig(requestConfig);
    return builder.build();
  }
}
