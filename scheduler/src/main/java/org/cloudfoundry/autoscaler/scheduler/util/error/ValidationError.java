package org.cloudfoundry.autoscaler.scheduler.util.error;

import java.util.Arrays;

class ValidationError {

  private Object object;
  private String errorMessageCode;
  private Object[] errorMessageArguments;

  ValidationError(Object object, Object[] errorMessageArguments, String errorMessageCode) {
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

  String getErrorMessageCode() {
    return errorMessageCode;
  }

  Object[] getErrorMessageArguments() {
    return errorMessageArguments;
  }

  @Override
  public String toString() {
    return "ValidationError [object="
        + object
        + ", errorMessageCode="
        + errorMessageCode
        + ", errorMessageArguments="
        + Arrays.toString(errorMessageArguments)
        + "]";
  }
}
