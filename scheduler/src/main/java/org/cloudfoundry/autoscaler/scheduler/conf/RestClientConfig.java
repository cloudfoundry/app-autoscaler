package org.cloudfoundry.autoscaler.scheduler.conf;

import java.nio.file.Path;
import javax.net.ssl.SSLContext;
import org.apache.hc.client5.http.classic.HttpClient;
import org.apache.hc.client5.http.config.ConnectionConfig;
import org.apache.hc.client5.http.config.RequestConfig;
import org.apache.hc.client5.http.impl.classic.HttpClientBuilder;
import org.apache.hc.client5.http.impl.io.PoolingHttpClientConnectionManagerBuilder;
import org.apache.hc.client5.http.io.HttpClientConnectionManager;
import org.apache.hc.client5.http.ssl.DefaultClientTlsStrategy;
import org.apache.hc.client5.http.ssl.HttpsSupport;
import org.apache.hc.client5.http.ssl.TlsSocketStrategy;
import org.apache.hc.core5.util.Timeout;
import org.cloudfoundry.autoscaler.scheduler.util.CfTrustStore;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.ssl.SslBundle;
import org.springframework.boot.ssl.SslBundles;
import org.springframework.boot.ssl.SslStoreBundle;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.http.client.ClientHttpRequestFactory;
import org.springframework.http.client.HttpComponentsClientHttpRequestFactory;
import org.springframework.util.StringUtils;
import org.springframework.web.client.RestOperations;
import org.springframework.web.client.RestTemplate;

@Configuration
public class RestClientConfig {
  private final SSLContext sslContext;

  @Autowired
  public RestClientConfig(
      SslBundles sslBundles,
      @Value("${client.ssl.protocol}") String protocol,
      @Value("${CF_CA_CERTS:}") String cfCaCerts) {
    SslBundle bundle = sslBundles.getBundle("scalingengine");
    SslStoreBundle stores = bundle.getStores();
    var trustStore =
        CfTrustStore.compose(
            stores.getTrustStore(), StringUtils.hasText(cfCaCerts) ? Path.of(cfCaCerts) : null);
    var clientStores =
        SslStoreBundle.of(stores.getKeyStore(), stores.getKeyStorePassword(), trustStore);
    this.sslContext =
        SslBundle.of(clientStores, bundle.getKey(), bundle.getOptions(), protocol)
            .createSslContext();
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
    TlsSocketStrategy tlsSocketStrategy =
        new DefaultClientTlsStrategy(
            this.sslContext,
            new String[] {protocol},
            null,
            null,
            HttpsSupport.getDefaultHostnameVerifier());

    var connectionConfig =
        ConnectionConfig.custom().setConnectTimeout(Timeout.ofSeconds(httpClientTimeout)).build();
    HttpClientConnectionManager ccm =
        PoolingHttpClientConnectionManagerBuilder.create()
            .setTlsSocketStrategy(tlsSocketStrategy)
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
