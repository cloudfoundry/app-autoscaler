package org.cloudfoundry.autoscaler.servicebroker.exception;

public class ProxyInitilizedFailedException extends Exception {
	
	private static final long serialVersionUID = 1L;

	public ProxyInitilizedFailedException() {
		super();
	}

	public ProxyInitilizedFailedException(String message) {
		super(message);
	}

	public ProxyInitilizedFailedException(Throwable cause) {
		super(cause);
	}
}
