package org.cloudfoundry.autoscaler.servicebroker.data.storeservice.couchdb;

import org.apache.http.auth.AuthScope;
import org.apache.http.auth.UsernamePasswordCredentials;
import org.apache.http.client.HttpClient;
import org.apache.http.conn.ClientConnectionManager;
import org.apache.http.impl.client.DecompressingHttpClient;
import org.apache.http.impl.client.DefaultHttpClient;
import org.apache.http.params.HttpParams;
import org.apache.log4j.Logger;
import org.ektorp.http.PreemptiveAuthRequestInterceptor;
import org.ektorp.http.StdHttpClient;


public class  HTTPProxyStdHttpClientBuilder extends StdHttpClient.Builder {

	private static final Logger logger = Logger.getLogger(HTTPProxyStdHttpClientBuilder.class);
	private String proxyUsername;
	private String proxyPassword;
	
    public HTTPProxyStdHttpClientBuilder proxyUsername(String proxyUsername)
    {
    	this.proxyUsername = proxyUsername;
        return this;
    }

    public HTTPProxyStdHttpClientBuilder proxyPassword(String proxyPassword)
    {
    	this.proxyPassword = proxyPassword;
        return this;
    }
    

    @Override
    public HttpClient configureClient() {
    	logger.info("using proxy client");		
    	HttpClient httpclient = super.configureClient();
    	
    	if (httpclient instanceof DecompressingHttpClient) {
			ClientConnectionManager connectionManager = httpclient.getConnectionManager();
			HttpParams params = httpclient.getParams();
			DefaultHttpClient client = new DefaultHttpClient(connectionManager, params);
			client.getCredentialsProvider().setCredentials(
					new AuthScope(host, port, AuthScope.ANY_REALM),
					new UsernamePasswordCredentials(username, password));
			client.addRequestInterceptor(
					new PreemptiveAuthRequestInterceptor(), 0);
			((DefaultHttpClient) client).getCredentialsProvider().setCredentials(
	    			new AuthScope(proxy, proxyPort,AuthScope.ANY_REALM),
					new UsernamePasswordCredentials(proxyUsername, proxyPassword));		
			return new DecompressingHttpClient(client);
		} else {
			AuthScope scope = new AuthScope(proxy, proxyPort,AuthScope.ANY_REALM);
			UsernamePasswordCredentials credential = new UsernamePasswordCredentials(proxyUsername, proxyPassword);
			((DefaultHttpClient) httpclient).getCredentialsProvider().setCredentials(scope,credential);		
			return httpclient;
		}
		
	}
  
    
}
