package org.cloudfoundry.autoscaler.exceptions;

public class TriggerNotFoundException extends Exception{
	public final static String ERROR_CODE = "TRIGGER_NOT_FOUND";
	public TriggerNotFoundException(String message) {
		super(message);
	}

	private static final long serialVersionUID = 5773025544116186465L;


}
