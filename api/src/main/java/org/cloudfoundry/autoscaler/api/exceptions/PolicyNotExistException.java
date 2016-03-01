package org.cloudfoundry.autoscaler.api.exceptions;

public class PolicyNotExistException extends Exception{
	private static final long serialVersionUID = 1L;

	private String appId;
	
	public PolicyNotExistException(String id) {
		super();
		appId = id;
	}
	  
	public PolicyNotExistException(String id, String message) {
		super(message);
		appId = id;
	}
	
	public PolicyNotExistException(String id, Throwable cause) {
		super(cause);
		appId = id;
	}
 
	public String getAppId() {
		return appId;
	}
}
