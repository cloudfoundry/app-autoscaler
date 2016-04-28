package org.cloudfoundry.autoscaler.common;

public class InputJSONParseErrorException extends Exception{
	private static final long serialVersionUID = 1L;

	private String context;
	
	public InputJSONParseErrorException(String context) {
		super();
		this.context = context;
	}
	  
	public InputJSONParseErrorException(String context, String message) {
		super(message);
		this.context = context;
	}
	
	public InputJSONParseErrorException(String context, Throwable cause) {
		super(cause);
		this.context = context;
	}
 
	public String getContext() {
		return context;
	}
}
