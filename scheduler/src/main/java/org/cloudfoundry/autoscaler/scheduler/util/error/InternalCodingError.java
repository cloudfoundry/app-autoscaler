package org.cloudfoundry.autoscaler.scheduler.util.error;

/**
 * An exception thrown when coding errors that can't be avoided by design occur at runtime.
 *
 * <p>This exception will result in a dedicated error page that indicates some coding has been done
 * wrong.
 *
 * <p>An example, when a method needs to assume a caller is going to provide certain information,
 * and the caller doesn't, especially when that information is sourced from posted information.
 */
public class InternalCodingError extends RuntimeException {

  private static final long serialVersionUID = 1L;

  /**
   * @param message - the message indicating the application coding problem that has occurred
   */
  public InternalCodingError(String message) {
    this(message, null);
  }

  /**
   * @param message - the message
   * @param cause - the causing exception
   */
  public InternalCodingError(String message, Throwable cause) {
    super(message, cause);
  }
}
