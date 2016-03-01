package org.cloudfoundry.autoscaler.cloudservice.manager.exceptions;


public class CloudLoginException extends CloudException
{

	private static final long serialVersionUID = 1L;

	public CloudLoginException() {
		super();
	}
	  
	public CloudLoginException(String message) {
		super(message);
	}
	  
	public CloudLoginException(Throwable cause) {
		super(cause);
	}
	
}
