package org.cloudfoundry.autoscaler.api.exceptions;


public class PolicyExistsException extends Exception
{

	private static final long serialVersionUID = 1L;

	private String configId;
	
	public PolicyExistsException(String id) {
		super();
		configId = id;
	}
	  
	public PolicyExistsException(String id, String message) {
		super(message);
		configId = id;
	}
	  
	public PolicyExistsException(String id, Throwable cause) {
		super(cause);
		configId = id;
	}
	
	public String getConfigId() {
		return configId;
	}
	
}
