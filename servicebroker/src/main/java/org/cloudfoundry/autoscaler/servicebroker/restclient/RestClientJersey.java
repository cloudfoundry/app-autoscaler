package org.cloudfoundry.autoscaler.servicebroker.restclient;

import java.io.IOException;
import java.net.Authenticator;
import java.net.HttpURLConnection;
import java.net.InetSocketAddress;
import java.net.PasswordAuthentication;
import java.net.Proxy;
import java.net.URL;
import java.security.SecureRandom;
import java.security.cert.CertificateException;
import java.util.Map;

import javax.net.ssl.HostnameVerifier;
import javax.net.ssl.HttpsURLConnection;
import javax.net.ssl.SSLContext;
import javax.net.ssl.TrustManager;
import javax.net.ssl.X509TrustManager;
import javax.ws.rs.core.Cookie;
import javax.ws.rs.core.MediaType;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.servicebroker.Constants;
import org.cloudfoundry.autoscaler.servicebroker.exception.ProxyInitilizedFailedException;
import org.cloudfoundry.autoscaler.servicebroker.mgr.ConfigManager;

import com.sun.jersey.api.client.Client;
import com.sun.jersey.api.client.WebResource.Builder;
import com.sun.jersey.api.client.config.ClientConfig;
import com.sun.jersey.api.client.config.DefaultClientConfig;
import com.sun.jersey.client.apache4.config.DefaultApacheHttpClient4Config;
import com.sun.jersey.client.urlconnection.HttpURLConnectionFactory;
import com.sun.jersey.client.urlconnection.URLConnectionClientHandler;

public class RestClientJersey implements IRestClient {

	private static final Logger logger = Logger.getLogger(RestClientJersey.class);
	private static final RestClientJersey instance = new RestClientJersey();

	private static Client  client = null;
	//	private static Client httpsRESTClient = null;
	//	private static Client httpRESTClient = null;

	private RestClientJersey() {
	}

	public static RestClientJersey getInstance() {
		return instance;
	}

	private ClientConfig getDefaultRestClientConfig() {
		ClientConfig config = new DefaultClientConfig();
		return getDefaultRestClientConfig(config);
	}

	private ClientConfig getDefaultRestClientConfig(ClientConfig config) {
		if (config == null) 
			config = new DefaultClientConfig();

		config.getProperties().put(DefaultApacheHttpClient4Config.PROPERTY_CONNECT_TIMEOUT,
				ConfigManager.getInt("connectionTimeout",100000));
		config.getProperties().put(DefaultApacheHttpClient4Config.PROPERTY_READ_TIMEOUT, 
				ConfigManager.getInt("connectionTimeout",100000));
		return config;
	}


	private Client getRestClient() throws ProxyInitilizedFailedException {
		if (client == null) {
			synchronized (this) {
				if (client == null) {
					URLConnectionClientHandler cc = new URLConnectionClientHandler(new ConnectionFactory());
					client = new Client(cc, getDefaultRestClientConfig());
				}
			}
		}    	
		return client;
	}


	@Override
	public IRestResourceBuilder resource(String url) {
		//Client client = isHTTPS(url)? getHTTPSRestClient():getHTTPRestClient();
		//return new IRestResourceBuilder(client, url);
		Client client = null;
		try {
			client = getRestClient();
		} catch (ProxyInitilizedFailedException e) {
			logger.error("Fail to init proxy information", e);
		}
		return new IRestResourceBuilder(client, url);
	}

	private class IRestResourceBuilder implements IRestClient.IRestResourceBuilder{

		private Builder builder;

		private IRestResourceBuilder(Client client, String url){
			builder = client.resource(url).getRequestBuilder();
		}

		@Override
		public IRestClient.IRestResourceBuilder header(
				String name, Object value) {
			builder.header(name, value);
			return this;
		}

		@Override
		public IRestClient.IRestResourceBuilder accept(
				MediaType mediaType) {
			builder.accept(mediaType);
			return this;
		}

		@Override
		public IRestClient.IRestResourceBuilder accept(
				String mediaType) {
			builder.accept(mediaType);
			return this;
		}

		@Override
		public IRestClient.IRestResourceBuilder type(
				MediaType mediaType) {
			builder.type(mediaType);
			return this;
		}

		@Override
		public IRestClient.IRestResourceBuilder type(
				String mediaType) {
			builder.type(mediaType);
			return this;
		}

		@Override
		public IRestClient.IRestResourceBuilder cookie(
				Cookie cookie) {

			builder.cookie(cookie);
			return this;
		}

		@Override
		public <T> T get(Class<T> c) {
			return builder.get(c);
		}

		@Override
		public <T> T post(Class<T> c) {
			return builder.post(c);

		}

		@Override
		public <T> T post(Class<T> c, Object requestEntity ) {
			return builder.post(c, requestEntity);

		}

		@Override
		public <T> T put(Class<T> c) {
			return builder.put(c);
		}

		@Override
		public <T> T put(Class<T> c, Object requestEntity) {
			return builder.put(c, requestEntity);
		}

		@Override
		public <T> T delete(Class<T> c) {
			return builder.delete(c);
		}


	}

	private class ConnectionFactory implements HttpURLConnectionFactory {

		Proxy proxy = null;
		String proxyUsername = null;
		String proxyPassword = null;

		public ConnectionFactory() {
		}

		public ConnectionFactory(String proxyHost, Integer proxyPort) {
			proxy = new Proxy(Proxy.Type.HTTP, new InetSocketAddress(proxyHost, proxyPort));
		}

		public ConnectionFactory(String proxyHost, Integer proxyPort, final String proxyUsername, final String proxyPassword) {
			this.proxyUsername = proxyUsername;
			this.proxyPassword = proxyPassword;
			proxy = new Proxy(Proxy.Type.HTTP, new InetSocketAddress(proxyHost, proxyPort));	    		
			Authenticator authenticator = new Authenticator() {
				public PasswordAuthentication getPasswordAuthentication() {
					return (new PasswordAuthentication(proxyUsername, proxyPassword.toCharArray()));
				}
			};
			Authenticator.setDefault(authenticator);	        
		}

		@Override
		public HttpURLConnection getHttpURLConnection(URL url) throws IOException {

			HttpURLConnection urlConnection = null;

			if ("https".equals(url.getProtocol())) {
				if (proxy!=null) {
					urlConnection = (HttpsURLConnection) url.openConnection(proxy);
				} else {
					urlConnection = (HttpsURLConnection) url.openConnection();
				}
				((HttpsURLConnection) urlConnection).setHostnameVerifier(getHostnameVerifier());
				((HttpsURLConnection) urlConnection).setSSLSocketFactory(getSSLContext().getSocketFactory());
				logger.info("setup SSL contxt here");

			} else {
				if (proxy!=null) {
					urlConnection = (HttpURLConnection) url.openConnection(proxy);
				} else {
					urlConnection = (HttpURLConnection) url.openConnection();
				}
			}
			return urlConnection;

		}

		private SSLContext getSSLContext() {
			TrustManager[] trustAllCerts = new TrustManager[] { new X509TrustManager() {
				public java.security.cert.X509Certificate[] getAcceptedIssuers() {
					return null;
				}

				public void checkClientTrusted(
						java.security.cert.X509Certificate[] arg0, String arg1)
								throws CertificateException {
				}

				public void checkServerTrusted(
						java.security.cert.X509Certificate[] arg0, String arg1)
								throws CertificateException {
				}
			} };

			SSLContext context = null;
			try {
				context = SSLContext.getInstance("TLS");
				context.init(null, trustAllCerts, new SecureRandom());
			} catch (java.security.GeneralSecurityException ex) {
			}
			return context;
		}

		private HostnameVerifier getHostnameVerifier() {
			return new HostnameVerifier() {
				@Override
				public boolean verify(String hostname, javax.net.ssl.SSLSession sslSession) {
					return true;
				}
			};
		}    

	}	 

}
