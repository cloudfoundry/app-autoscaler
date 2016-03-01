package org.cloudfoundry.autoscaler.cloudservice.manager.exceptions;


public class CloudNotLoggedInException extends CloudException
{

	private static final long serialVersionUID = 1L;

	public CloudNotLoggedInException() {
		super();
	}
	  
	public CloudNotLoggedInException(String message) {
		super(message);
	}
	  
	public CloudNotLoggedInException(Throwable cause) {
		super(cause);
	}
	
}
