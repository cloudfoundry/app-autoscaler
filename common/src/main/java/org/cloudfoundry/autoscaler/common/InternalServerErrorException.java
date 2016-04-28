package org.cloudfoundry.autoscaler.common;

public class InternalServerErrorException extends Exception{
	private static final long serialVersionUID = 1L;

	private String context="";
	
	public InternalServerErrorException(String context) {
		super();
		this.context = context;
	}
	  
	public InternalServerErrorException(String context, String message) {
		super(message);
		this.context = context;
	}
	
	public InternalServerErrorException(String context, Throwable cause) {
		super(cause);
		this.context = context;
	}
 
	public String getContext() {
		return context;
	}

}
