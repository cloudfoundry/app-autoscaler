package org.cloudfoundry.autoscaler.common;

public class AppInfoNotFoundException extends Exception {
	private static final long serialVersionUID = 1L;

	private String appId;
	
	public AppInfoNotFoundException(String id) {
		super();
		appId = id;
	}
	  
	public AppInfoNotFoundException(String id, String message) {
		super(message);
		appId = id;
	}
	
	public AppInfoNotFoundException(String id, Throwable cause) {
		super(cause);
		appId = id;
	}
 
	public String getAppId() {
		return appId;
	}
}
