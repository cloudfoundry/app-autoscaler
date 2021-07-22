package org.cloudfoundry.autoscaler.scheduler.util.error;

public class SchedulerInternalException extends RuntimeException {

  private static final long serialVersionUID = 1L;

  public SchedulerInternalException() {
    super();
  }

  public SchedulerInternalException(
      String message, Throwable cause, boolean enableSuppression, boolean writableStackTrace) {
    super(message, cause, enableSuppression, writableStackTrace);
  }

  public SchedulerInternalException(String message, Throwable cause) {
    super(message, cause);
  }

  public SchedulerInternalException(String message) {
    super(message);
  }

  public SchedulerInternalException(Throwable cause) {
    super(cause);
  }
}
