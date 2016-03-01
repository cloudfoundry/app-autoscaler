package org.cloudfoundry.autoscaler.api.exceptions;


public class AutoScalerException extends Exception
{

	private static final long serialVersionUID = 1L;

	public AutoScalerException() {
		super();
	}
	  
	public AutoScalerException(String message) {
		super(message);
	}
	  
	public AutoScalerException(Throwable cause) {
		super(cause);
	}
	
}
