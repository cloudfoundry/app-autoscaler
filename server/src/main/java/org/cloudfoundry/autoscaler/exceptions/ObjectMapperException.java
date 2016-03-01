package org.cloudfoundry.autoscaler.exceptions;


// by extending RuntimeException we don't have to declare this exception 
public class ObjectMapperException extends RuntimeException
{

	private static final long serialVersionUID = 1L;

	public ObjectMapperException() {
		super();
	}
	  
	public ObjectMapperException(String message) {
		super(message);
	}
	  
	public ObjectMapperException(Throwable cause) {
		super(cause);
	}
	
}
