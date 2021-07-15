package org.cloudfoundry.autoscaler.scheduler.util.error;

public class InvalidDataException extends RuntimeException {

  private static final long serialVersionUID = 1L;

  public InvalidDataException() {
    super();
  }

  public InvalidDataException(
      String message, Throwable cause, boolean enableSuppression, boolean writableStackTrace) {
    super(message, cause, enableSuppression, writableStackTrace);
  }

  public InvalidDataException(String message, Throwable cause) {
    super(message, cause);
  }

  public InvalidDataException(String message) {
    super(message);
  }

  public InvalidDataException(Throwable cause) {
    super(cause);
  }
}
