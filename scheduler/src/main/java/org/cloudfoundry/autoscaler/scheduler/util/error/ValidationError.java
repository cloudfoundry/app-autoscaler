package org.cloudfoundry.autoscaler.scheduler.util.error;

import java.util.Arrays;

/**
 * 
 *
 */
public class ValidationError {

	private Object object;
	private String errorMessageCode;
	private Object[] errorMessageArguments;

	public ValidationError(Object object, Object[] errorMessageArguments, String errorMessageCode) {
		this.object = object;
		this.errorMessageCode = errorMessageCode;
		this.errorMessageArguments = errorMessageArguments;
	}

	public Object getObject() {
		return object;
	}

	public void setObject(Object object) {
		this.object = object;
	}

	public String getErrorMessageCode() {
		return errorMessageCode;
	}

	public void setErrorMessageCode(String errorMessageCode) {
		this.errorMessageCode = errorMessageCode;
	}

	public Object[] getErrorMessageArguments() {
		return errorMessageArguments;
	}

	public void setErrorMessageArguments(Object[] errorMessageArguments) {
		this.errorMessageArguments = errorMessageArguments;
	}

	@Override
	public String toString() {
		return "ValidationError [object=" + object + ", errorMessageCode=" + errorMessageCode
				+ ", errorMessageArguments=" + Arrays.toString(errorMessageArguments) + "]";
	}

}
