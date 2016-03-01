package org.cloudfoundry.autoscaler.api.exceptions;

public class BssServiceException extends Exception{

	/**
	 * 
	 */
	private static final long serialVersionUID = 9041847950962322562L;
	
	public BssServiceException(String message){
		super(message);
	}
}
