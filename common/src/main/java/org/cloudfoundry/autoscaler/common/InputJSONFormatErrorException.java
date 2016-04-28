package org.cloudfoundry.autoscaler.common;

public class InputJSONFormatErrorException extends Exception{
	private static final long serialVersionUID = 1L;

	private String context;
	
	private int line = 0;
	
	private int column = 0;
	
	public InputJSONFormatErrorException(String context) {
		super();
		this.context = context;
	}
	  
	public InputJSONFormatErrorException(String context, String message) {
		super(message);
		this.context = context;
	}
	
	public InputJSONFormatErrorException(String context, String message, int line, int column) {
		super(message);
		this.context = context;
		this.line = line;
		this.column = column;
	}
	
	public InputJSONFormatErrorException(String context, Throwable cause) {
		super(cause);
		this.context = context;
	}
    
	public InputJSONFormatErrorException(InputJSONFormatErrorException e) {
		super(e.getMessage());
		this.context = e.getContext();
	}
	
	public String getContext() {
		return context;
	}
	
	public int getLine() {
		return line;
	}
	
	public int getColumn() {
		return column;
	}
}
