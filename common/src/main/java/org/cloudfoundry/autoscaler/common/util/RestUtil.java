package org.cloudfoundry.autoscaler.common.util;


import java.security.SecureRandom;
import java.security.cert.CertificateException;

import javax.net.ssl.HostnameVerifier;
import javax.net.ssl.HttpsURLConnection;
import javax.net.ssl.SSLContext;
import javax.net.ssl.SSLSession;
import javax.net.ssl.TrustManager;
import javax.net.ssl.X509TrustManager;
import javax.ws.rs.core.Response;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.common.util.ConfigManager;
import org.json.JSONObject;

import com.sun.jersey.api.client.Client;
import com.sun.jersey.api.client.config.DefaultClientConfig;
import com.sun.jersey.client.apache4.config.DefaultApacheHttpClient4Config;
import com.sun.jersey.client.urlconnection.HTTPSProperties;

public class RestUtil {
	
	private static final Logger logger = Logger.getLogger(RestUtil.class);
	private static volatile Client httpsRESTClient = null;
	private static volatile Client httpRESTClient = null;

    public static Response getErrorResponse(final Object text) {
        JSONObject response = new JSONObject();
        response.put("description", text);
        return Response.status(499).entity(response.toString()).build();
    }
    
    public static Client getHTTPSRestClient (){
        if (httpsRESTClient == null) {
        	synchronized (RestUtil.class) {
        		if (httpsRESTClient == null) {
            		DefaultClientConfig config = setupRestClientWithTrustSelfSignedCert();
            		httpsRESTClient = Client.create(config);
        		}

        	}
        }    	
    	return httpsRESTClient;
    }
    
    public static Client getHTTPRestClient(){
        if (httpRESTClient == null) {
        	synchronized (RestUtil.class) {
        		if (httpRESTClient == null) {
        			httpRESTClient = Client.create();  
        		}
        	}
        }    	
    	return httpRESTClient;

    }
    
	private static DefaultClientConfig setupRestClientWithTrustSelfSignedCert() {
		TrustManager[] trustAllCerts = new TrustManager[] { new X509TrustManager() {
			public java.security.cert.X509Certificate[] getAcceptedIssuers() {
				return null;
			}

			@Override
			public void checkClientTrusted(
					java.security.cert.X509Certificate[] arg0, String arg1)
					throws CertificateException {

			}

			@Override
			public void checkServerTrusted(
					java.security.cert.X509Certificate[] arg0, String arg1)
					throws CertificateException {
			}
		} };

		SSLContext context;
		try {
			context = SSLContext.getInstance("TLS");
			context.init(null, trustAllCerts, new SecureRandom());

			HttpsURLConnection.setDefaultSSLSocketFactory(context
					.getSocketFactory());

			DefaultClientConfig config = new DefaultClientConfig();
			config.getProperties().put(
					HTTPSProperties.PROPERTY_HTTPS_PROPERTIES,
					new HTTPSProperties(new HostnameVerifier() {

						@Override
						public boolean verify(String arg0, SSLSession arg1) {
							return false;
						}

					}, context));
			config.getProperties().put(DefaultApacheHttpClient4Config.PROPERTY_CONNECT_TIMEOUT,
					ConfigManager.getInt("connectionTimeout",100000));
			config.getProperties()
					.put(DefaultApacheHttpClient4Config.PROPERTY_READ_TIMEOUT, ConfigManager.getInt("connectionTimeout",100000));
			
			return config;

		} catch (Exception e) {
			logger.error("Failed to setup SSL connection with target");
			return null;
		}
	}
    
    
	public static boolean isHTTPS(String url) {
		return url != null && url.startsWith("https");
	}
	


}

