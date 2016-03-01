package org.cloudfoundry.autoscaler.servicebroker.exception;

public class AlreadyBoundAnotherServiceException extends Exception {
	
	private static final long serialVersionUID = 1L;

	public AlreadyBoundAnotherServiceException() {
		super();
	}

	public AlreadyBoundAnotherServiceException(String message) {
		super(message);
	}

	public AlreadyBoundAnotherServiceException(Throwable cause) {
		super(cause);
	}
}
