package org.cloudfoundry.autoscaler.common;

public class OutputJSONParseErrorException extends Exception{
	private static final long serialVersionUID = 1L;

	private String context;
	
	public OutputJSONParseErrorException(String context) {
		super();
		this.context = context;
	}
	  
	public OutputJSONParseErrorException(String context, String message) {
		super(message);
		this.context = context;
	}
	
	public OutputJSONParseErrorException(String context, Throwable cause) {
		super(cause);
		this.context = context;
	}
 
	public String getContext() {
		return context;
	}
}
