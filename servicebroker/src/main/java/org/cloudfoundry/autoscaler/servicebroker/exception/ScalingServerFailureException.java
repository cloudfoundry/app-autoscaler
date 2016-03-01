package org.cloudfoundry.autoscaler.servicebroker.exception;

public class ScalingServerFailureException extends Exception {
	
	private static final long serialVersionUID = 1L;

	public ScalingServerFailureException() {
		super();
	}

	public ScalingServerFailureException(String message) {
		super(message);
	}

	public ScalingServerFailureException(Throwable cause) {
		super(cause);
	}
}
