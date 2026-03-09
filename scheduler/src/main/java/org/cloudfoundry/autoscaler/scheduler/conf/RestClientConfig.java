package org.cloudfoundry.autoscaler.scheduler.conf;

import java.security.KeyStore;
import javax.net.ssl.SSLContext;
import javax.net.ssl.TrustManagerFactory;
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

    // Load client certificate KeyManagers for mutual TLS authentication
    javax.net.ssl.KeyManager[] keyManagers = loadKeyManagers(sslBundle);

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
        logger.debug("No keystore configured in SSL bundle");
        return null;
      }

      if (keyStore.size() == 0) {
        logger.debug("Keystore is empty");
        return null;
      }

      logger.debug("Loaded keystore with {} key(s) from SSL bundle", keyStore.size());

      // Create KeyManager from the keystore
      javax.net.ssl.KeyManagerFactory kmf =
          javax.net.ssl.KeyManagerFactory.getInstance(
              javax.net.ssl.KeyManagerFactory.getDefaultAlgorithm());
      kmf.init(keyStore, null); // Password is null as it's already loaded by Spring

      return kmf.getKeyManagers();

    } catch (Exception e) {
      logger.warn(
          "Could not load keystore from SSL bundle, client certificate auth will not be available: {}",
          e.getMessage());
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
        logger.debug("No truststore configured in SSL bundle");
        return null;
      }

      // Check if the truststore actually has certificates
      if (trustStore.size() == 0) {
        logger.debug("Truststore is empty");
        return null;
      }

      logger.debug("Loaded truststore with {} certificate(s) from SSL bundle", trustStore.size());

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
