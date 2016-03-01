package org.cloudfoundry.autoscaler.api.exceptions;

public class NoAttachedPolicyException  extends Exception{
	/**
	 * 
	 */
	private static final long serialVersionUID = -2621595520272413555L;

	public NoAttachedPolicyException(){
		super();
	}
	public NoAttachedPolicyException(String message){
		super(message);
	}
}
