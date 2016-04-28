package org.cloudfoundry.autoscaler.common;

public class ClientIDLoginFailedException extends Exception{

	private static final long serialVersionUID = 1L;

	private String clientID ; 
	
	public ClientIDLoginFailedException(String id, String message) {
		super(message);
		clientID = id;
	}
	
	public String getClientID(){
		return clientID;
	}

}
