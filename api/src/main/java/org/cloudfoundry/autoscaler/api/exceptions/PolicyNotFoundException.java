package org.cloudfoundry.autoscaler.api.exceptions;


public class PolicyNotFoundException extends Exception
{

	private static final long serialVersionUID = 1L;

	private String policyName;
	
	public PolicyNotFoundException() {
		super();
	}
	
	public PolicyNotFoundException(String id) {
		super();
		policyName = id;
	}
	  
	public PolicyNotFoundException(String id, String message) {
		super(message);
		policyName = id;
	}
	
	public PolicyNotFoundException(String id, Throwable cause) {
		super(cause);
		policyName = id;
	}
	
	public String getPolicyId() {
		return policyName;
	}
	
}
