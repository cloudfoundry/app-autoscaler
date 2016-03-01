package org.cloudfoundry.autoscaler.exceptions;


public class AuthorizationException extends Exception
{

	private static final long serialVersionUID = 1L;

	private String accountId;
	
	public AuthorizationException(String id) {
		super();
		accountId = id;
	}
	  
	public AuthorizationException(String id, String message) {
		super(message);
		accountId = id;
	}
	
	public AuthorizationException(String id, Throwable cause) {
		super(cause);
		accountId = id;
	}
 
	public String getAccountId() {
		return accountId;
	}

}
