package org.cloudfoundry.autoscaler.api.exceptions;


public class AppListNotEmptyException extends Exception
{

	private static final long serialVersionUID = 1L;

	private String configId;
	
	public AppListNotEmptyException(String id) {
		super();
		configId = id;
	}
	  
	public AppListNotEmptyException(String id, String message) {
		super(message);
		configId = id;
	}
	
	public AppListNotEmptyException(String id, Throwable cause) {
		super(cause);
		configId = id;
	}
	
	public String getConfigId() {
		return configId;
	}
	
}
