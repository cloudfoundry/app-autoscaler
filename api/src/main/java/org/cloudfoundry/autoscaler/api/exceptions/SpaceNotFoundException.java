package org.cloudfoundry.autoscaler.api.exceptions;


public class SpaceNotFoundException extends Exception
{

	private static final long serialVersionUID = 1L;

	private String spaceId;
	
	public SpaceNotFoundException(String id) {
		super();
		spaceId = id;
	}
	  
	public SpaceNotFoundException(String id, String message) {
		super(message);
		spaceId = id;
	}
	
	public SpaceNotFoundException(String id, Throwable cause) {
		super(cause);
		spaceId = id;
	}
 
	public String getSpaceId() {
		return spaceId;
	}

}
