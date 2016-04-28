package org.cloudfoundry.autoscaler.common;

public class PolicyNotFoundException extends Exception{
	private static final long serialVersionUID = 1L;

	private String appId;
	
	public PolicyNotFoundException(String id) {
		super();
		appId = id;
	}
	  
	public PolicyNotFoundException(String id, String message) {
		super(message);
		appId = id;
	}
	
	public PolicyNotFoundException(String id, Throwable cause) {
		super(cause);
		appId = id;
	}
 
	public String getAppId() {
		return appId;
	}
}
