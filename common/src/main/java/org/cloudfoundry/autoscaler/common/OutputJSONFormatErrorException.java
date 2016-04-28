package org.cloudfoundry.autoscaler.common;

public class OutputJSONFormatErrorException extends Exception{
	private static final long serialVersionUID = 1L;

	private String context;
	
	public OutputJSONFormatErrorException(String context) {
		super();
		this.context = context;
	}
	  
	public OutputJSONFormatErrorException(String context, String message) {
		super(message);
		this.context = context;
	}
	
	public OutputJSONFormatErrorException(String context, Throwable cause) {
		super(cause);
		this.context = context;
	}
 
	public OutputJSONFormatErrorException(OutputJSONFormatErrorException e) {
		super(e.getMessage());
		this.context = e.getContext();
	}
	
	public String getContext() {
		return context;
	}
}
