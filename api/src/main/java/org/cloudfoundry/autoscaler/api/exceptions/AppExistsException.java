package org.cloudfoundry.autoscaler.api.exceptions;


public class AppExistsException extends Exception
{

	private static final long serialVersionUID = 1L;

	private String appId;
	
	public AppExistsException(String id) {
		super();
		appId = id;
	}
	  
	public AppExistsException(String id, String message) {
		super(message);
		appId = id;
	}
	  
	public AppExistsException(String id, Throwable cause) {
		super(cause);
		appId = id;
	}
	
	public String getAppId() {
		return appId;
	}

}
