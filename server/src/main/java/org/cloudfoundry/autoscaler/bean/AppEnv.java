package org.cloudfoundry.autoscaler.bean;
/**
 * Application environment in CF
 * 
 *
 */
public class AppEnv {
	String[] application_uris;

	public String[] getApplication_uris() {
		return application_uris.clone();
	}

	public void setApplication_uris(String[] application_uris) {
		this.application_uris = application_uris.clone();
	}
	
}
