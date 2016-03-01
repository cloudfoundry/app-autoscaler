package org.cloudfoundry.autoscaler.servicebroker.exception;

public class ServerUrlMappingNotFoundException extends Exception {
	
	private static final long serialVersionUID = 1L;

	public ServerUrlMappingNotFoundException() {
		super();
	}

	public ServerUrlMappingNotFoundException(String message) {
		super(message);
	}

	public ServerUrlMappingNotFoundException(Throwable cause) {
		super(cause);
	}
}
