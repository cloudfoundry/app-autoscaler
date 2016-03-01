package org.cloudfoundry.autoscaler.cloudservice.manager.exceptions;

public class AppNotFoundException extends Exception{

	/**
	 * 
	 */
	private static final long serialVersionUID = 478256206624809848L;
	public AppNotFoundException (String message){
		super(message);
	}

}
