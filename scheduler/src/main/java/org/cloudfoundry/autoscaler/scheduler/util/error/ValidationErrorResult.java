package org.cloudfoundry.autoscaler.scheduler.util.error;

import java.util.ArrayList;
import java.util.List;

import org.quartz.SchedulerException;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Configurable;
import org.springframework.stereotype.Component;
import org.springframework.web.context.annotation.RequestScope;

/**
 * Request scoped object for tracking results of validation (business rules validation mostly).
 *
 */

@Component
@Configurable
@RequestScope
public class ValidationErrorResult {

	@Autowired
	MessageBundleResourceHelper messageBundleResourceHelper;
	boolean initialised;
	private List<ValidationError> errorList; // NOTE:Leave error list null until, have actual errors

	public ValidationErrorResult() {
	}

	/**
	 * Initialise an instance with an empty errors list and no association with a specific object type
	 * @param objectName
	 */
	public void initEmpty() {
		errorList = null;
		initialised = true;
	}

	private void checkInitialised() {
		if (!initialised) {
			throw new InternalCodingError("ValidationResult not initialised - call  initEmpty");
		}
	}

	public void addFieldError(Object object, String messageCode, Object... arguments) {

		internalAddError(new ValidationError(object, arguments, messageCode));
	}

	public void addErrorForDatabaseValidationException(DatabaseValidationException dve, String messageCode,
			Object... arguments) {
		addFieldError(dve, messageCode, arguments);

	}

	public void addErrorForQuartzSchedulerException(SchedulerException se, String messageCode, Object... arguments) {
		addFieldError(se, messageCode, arguments);

	}

	private void internalAddError(ValidationError error) {
		checkInitialised();
		if (errorList == null) {
			errorList = new ArrayList<ValidationError>();
		}
		errorList.add(error);
	}

	/**
	 * A list of error messages corresponding to the errors contained in this instance
	 * @return a List<String> containing the error messages.
	 */
	public List<String> getAllErrorMessages() {

		if (errorList == null || errorList.size() == 0) {
			return new ArrayList<String>();
		}

		List<String> errorMessages = new ArrayList<String>(errorList.size());

		for (ValidationError error : errorList) {

			String resourceKey = error.getErrorMessageCode();
			Object[] messageArguments = error.getErrorMessageArguments();

			// Lookup error with arguments
			String errorMessage = messageBundleResourceHelper.lookupMessage(resourceKey, messageArguments);

			errorMessages.add(errorMessage);
		}

		return errorMessages;
	}

	/**
	 * @return - true if this instance contains any errors
	 */
	public boolean hasErrors() {
		checkInitialised();
		return errorList != null && errorList.size() > 0;
	}

}
