package org.cloudfoundry.autoscaler.api.exceptions;

public class NoMonitorServiceBoundException extends Exception {

	/**
	 * 
	 */
	private static final long serialVersionUID = 3559499685006342538L;
	
	public NoMonitorServiceBoundException(String message){
		super(message);
	}
	
	public NoMonitorServiceBoundException(){
		
	}

}
