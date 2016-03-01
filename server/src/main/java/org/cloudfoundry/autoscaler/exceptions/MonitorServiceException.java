package org.cloudfoundry.autoscaler.exceptions;

public class MonitorServiceException extends Exception{
	/**
	 * 
	 */
	private static final long serialVersionUID = 2200915241969976777L;
	private int errorCode = 0;
	public MonitorServiceException(int code, String message){
		super(message);
		errorCode = code;
	}

	public MonitorServiceException(String message){
		super(message);
	}
	
	public MonitorServiceException(String message, Exception e){
		super(message,e);
	}
	
	public int getErrorCode(){
		return errorCode;
	}
}
