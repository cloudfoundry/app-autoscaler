package org.cloudfoundry.autoscaler.scheduler.util.error;

public class DatabaseValidationException extends RuntimeException {

  private static final long serialVersionUID = 1L;

  public DatabaseValidationException() {
    super();
  }

  public DatabaseValidationException(String message) {
    super(message);
  }

  public DatabaseValidationException(Throwable cause) {
    super(cause);
  }

  public DatabaseValidationException(String message, Throwable cause) {
    super(message, cause);
  }
}
