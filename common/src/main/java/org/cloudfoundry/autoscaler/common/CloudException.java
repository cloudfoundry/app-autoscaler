package org.cloudfoundry.autoscaler.common;

public class CloudException extends Exception{

	private static final long serialVersionUID = 1L;
	private String errorCode;

	public CloudException() {
		super();
	}

	public CloudException(String code, String message) {
		super(message);
		this.errorCode = code;
	}
	
	public CloudException(String message) {
		super(message);
	}
	  
	public CloudException(Throwable cause) {
		super(cause);
	}
	
	public String getErrorCode(){
		return this.errorCode;
	}
}
