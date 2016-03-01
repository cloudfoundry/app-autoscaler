package org.cloudfoundry.autoscaler.api.exceptions;


public class AppNotFoundException extends Exception
{

	private static final long serialVersionUID = 1L;

	private String appId;
	
	public AppNotFoundException(String id) {
		super();
		appId = id;
	}
	  
	public AppNotFoundException(String id, String message) {
		super(message);
		appId = id;
	}
	
	public AppNotFoundException(String id, Throwable cause) {
		super(cause);
		appId = id;
	}
 
	public String getAppId() {
		return appId;
	}

}
