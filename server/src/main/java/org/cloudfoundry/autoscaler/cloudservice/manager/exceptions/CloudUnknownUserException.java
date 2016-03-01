package org.cloudfoundry.autoscaler.cloudservice.manager.exceptions;


public class CloudUnknownUserException extends CloudException
{

	private static final long serialVersionUID = 1L;
	
	private String userOrg;
	private String userSpace;

	public CloudUnknownUserException(String org, String space) {
		super();
		userOrg   = org;
		userSpace = space;
	}
	  
	public CloudUnknownUserException(String org, String space, String message) {
		super(message);
		userOrg   = org;
		userSpace = space;
	}
	  
	public CloudUnknownUserException(String org, String space, Throwable cause) {
		super(cause);
		userOrg   = org;
		userSpace = space;
	}

	public String getOrg() {
		return userOrg;
	}

	public String getSpace() {
		return userSpace;
	}

}
