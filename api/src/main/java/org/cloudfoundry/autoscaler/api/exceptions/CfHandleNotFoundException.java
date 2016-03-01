package org.cloudfoundry.autoscaler.api.exceptions;


public class CfHandleNotFoundException extends Exception
{

	private static final long serialVersionUID = 1L;

	private String handleId;
	
	public CfHandleNotFoundException(String id) {
		super();
		handleId = id;
	}
	  
	public CfHandleNotFoundException(String id, String message) {
		super(message);
		handleId = id;
	}
	
	public CfHandleNotFoundException(String id, Throwable cause) {
		super(cause);
		handleId = id;
	}
	
	public String getHandleId() {
		return handleId;
	}
	
}
