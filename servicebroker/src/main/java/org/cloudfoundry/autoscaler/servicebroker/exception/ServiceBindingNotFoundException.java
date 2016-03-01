package org.cloudfoundry.autoscaler.servicebroker.exception;

public class ServiceBindingNotFoundException extends Exception {
	
	private static final long serialVersionUID = 1L;

	public ServiceBindingNotFoundException() {
		super();
	}

	public ServiceBindingNotFoundException(String message) {
		super(message);
	}

	public ServiceBindingNotFoundException(Throwable cause) {
		super(cause);
	}
}
