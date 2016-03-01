package org.cloudfoundry.autoscaler.api.exceptions;


public class OrgNotFoundException extends Exception
{

	private static final long serialVersionUID = 1L;

	private String orgId;
	
	public OrgNotFoundException(String id) {
		super();
		orgId = id;
	}
	  
	public OrgNotFoundException(String id, String message) {
		super(message);
		orgId = id;
	}
	
	public OrgNotFoundException(String id, Throwable cause) {
		super(cause);
		orgId = id;
	}
 
	public String getSpaceId() {
		return orgId;
	}

}
