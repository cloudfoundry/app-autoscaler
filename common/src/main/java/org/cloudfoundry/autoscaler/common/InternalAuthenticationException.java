package org.cloudfoundry.autoscaler.common;

public class InternalAuthenticationException extends Exception{

	private static final long serialVersionUID = 1L;

	private String context;
	
	public InternalAuthenticationException(String context) {
		super();
		this.context = context;
	}
	  
	public InternalAuthenticationException(String context, String message) {
		super(message);
		this.context = context;
	}
	
	public InternalAuthenticationException(String context, Throwable cause) {
		super(cause);
		this.context = context;
	}

	public InternalAuthenticationException(InternalAuthenticationException e) {
		super(e.getMessage());
		this.context = e.getContext();
	}
	
	public String getContext() {
		return context;
	}

}
