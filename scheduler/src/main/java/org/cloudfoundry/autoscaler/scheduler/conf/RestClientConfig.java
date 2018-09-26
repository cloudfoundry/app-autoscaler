package org.cloudfoundry.autoscaler.scheduler.conf;

import java.io.File;
import java.io.FileInputStream;
import java.security.KeyStore;

import javax.net.ssl.SSLContext;

import org.apache.http.client.HttpClient;
import org.apache.http.client.config.RequestConfig;
import org.apache.http.config.Registry;
import org.apache.http.config.RegistryBuilder;
import org.apache.http.conn.HttpClientConnectionManager;
import org.apache.http.conn.socket.ConnectionSocketFactory;
import org.apache.http.conn.socket.PlainConnectionSocketFactory;
import org.apache.http.conn.ssl.SSLConnectionSocketFactory;
import org.apache.http.impl.client.HttpClientBuilder;
import org.apache.http.impl.conn.PoolingHttpClientConnectionManager;
import org.apache.http.ssl.SSLContextBuilder;
import org.apache.http.ssl.SSLContexts;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.support.PropertySourcesPlaceholderConfigurer;
import org.springframework.http.client.ClientHttpRequestFactory;
import org.springframework.http.client.HttpComponentsClientHttpRequestFactory;
import org.springframework.web.client.RestOperations;
import org.springframework.web.client.RestTemplate;

@Configuration
public class RestClientConfig {
	@Bean
	public RestOperations restOperations(ClientHttpRequestFactory clientHttpRequestFactory) throws Exception {
		return new RestTemplate(clientHttpRequestFactory);
	}

	@Bean
	public ClientHttpRequestFactory clientHttpRequestFactory(HttpClient httpClient) {
		return new HttpComponentsClientHttpRequestFactory(httpClient);
	}

	@Bean
	public HttpClient httpClient(@Value("${client.ssl.key-store}") String keyStoreFile,
			@Value("${client.ssl.key-store-password}") String keyStorePassword,
			@Value("${client.ssl.key-store-type}") String keyStoreType,
			@Value("${client.ssl.trust-store}") String trustStoreFile,
			@Value("${client.ssl.trust-store-password}") String trustStorePassword,
			@Value("${client.ssl.protocol}") String protocol,
			@Value("${client.httpClientTimeout}") Integer httpClientTimeout)  throws Exception {
		KeyStore trustStore = KeyStore.getInstance(KeyStore.getDefaultType());
		KeyStore keyStore = KeyStore.getInstance(keyStoreType == null ? KeyStore.getDefaultType() : keyStoreType);

		try (FileInputStream trustStoreInstream = new FileInputStream(new File(trustStoreFile));
				FileInputStream keyStoreInstream = new FileInputStream(new File(keyStoreFile))) {
			trustStore.load(trustStoreInstream, trustStorePassword.toCharArray());
			keyStore.load(keyStoreInstream, keyStorePassword.toCharArray());
		}

		SSLContextBuilder sslCtxBuilder = SSLContexts.custom().loadTrustMaterial(trustStore, null);
		sslCtxBuilder = sslCtxBuilder.loadKeyMaterial(keyStore, keyStorePassword.toCharArray());

		SSLContext sslcontext = sslCtxBuilder.build();

		HttpClientBuilder builder = HttpClientBuilder.create();
		SSLConnectionSocketFactory sslsf = new SSLConnectionSocketFactory(sslcontext, new String[] { protocol }, null,
				SSLConnectionSocketFactory.getDefaultHostnameVerifier());

		builder.setSSLSocketFactory(sslsf);
		Registry<ConnectionSocketFactory> registry = RegistryBuilder.<ConnectionSocketFactory> create()
				.register("https", sslsf).register("http", new PlainConnectionSocketFactory()).build();
		HttpClientConnectionManager ccm = new PoolingHttpClientConnectionManager(registry);
		builder.setConnectionManager(ccm);
		RequestConfig requestConfig = RequestConfig.custom()
				  .setConnectTimeout(httpClientTimeout * 1000)
				  .setConnectionRequestTimeout(httpClientTimeout * 1000)
				  .setSocketTimeout(httpClientTimeout * 1000).build();
		builder.setDefaultRequestConfig(requestConfig);
		return builder.build();
	}

	@Bean
	public static PropertySourcesPlaceholderConfigurer propertySourcesPlaceholderConfigurer() {
		return new PropertySourcesPlaceholderConfigurer();
	}
}
