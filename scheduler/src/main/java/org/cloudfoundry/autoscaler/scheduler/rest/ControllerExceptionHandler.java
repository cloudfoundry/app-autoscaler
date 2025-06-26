package org.cloudfoundry.autoscaler.scheduler.rest;

import jakarta.servlet.http.HttpServletRequest;
import jakarta.validation.ConstraintViolationException;
import java.util.List;
import org.cloudfoundry.autoscaler.scheduler.util.error.InvalidDataException;
import org.cloudfoundry.autoscaler.scheduler.util.error.SchedulerInternalException;
import org.cloudfoundry.autoscaler.scheduler.util.error.ValidationErrorResult;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.MethodArgumentNotValidException;
import org.springframework.web.bind.MissingServletRequestParameterException;
import org.springframework.web.bind.annotation.ControllerAdvice;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.servlet.resource.NoResourceFoundException;

@ControllerAdvice
public class ControllerExceptionHandler {
  private Logger logger = LoggerFactory.getLogger(this.getClass());

  @Autowired private ValidationErrorResult validationErrorResult;

  @ExceptionHandler(Exception.class)
  public ResponseEntity<List<String>> handleException(HttpServletRequest req, Exception e) {
    logger.error("Internal Server Error", e);

    return new ResponseEntity<>(null, null, HttpStatus.INTERNAL_SERVER_ERROR);
  }

  @ExceptionHandler(MethodArgumentNotValidException.class)
  public ResponseEntity<List<String>> handleMethodArgumentNotValidException(
      HttpServletRequest req, Exception e) {

    return new ResponseEntity<>(null, null, HttpStatus.BAD_REQUEST);
  }

  @ExceptionHandler(InvalidDataException.class)
  public ResponseEntity<List<String>> handleValidationException(
      HttpServletRequest req, Exception e) {

    List<String> errors = validationErrorResult.getAllErrorMessages();
    return new ResponseEntity<>(errors, null, HttpStatus.BAD_REQUEST);
  }

  @ExceptionHandler(SchedulerInternalException.class)
  public ResponseEntity<List<String>> handleDatabaseValidationException(
      HttpServletRequest req, Exception e) {
    logger.error("Internal Server Error", e);

    List<String> errors = validationErrorResult.getAllErrorMessages();
    return new ResponseEntity<>(errors, null, HttpStatus.INTERNAL_SERVER_ERROR);
  }

  @ExceptionHandler(MissingServletRequestParameterException.class)
  @ResponseStatus(HttpStatus.BAD_REQUEST)
  private void handleMissingParameter(MissingServletRequestParameterException e) {}

  @ExceptionHandler(value = {ConstraintViolationException.class})
  protected ResponseEntity<Object> handleConstraintViolation(ConstraintViolationException e) {
    return new ResponseEntity<>(e.getMessage(), null, HttpStatus.BAD_REQUEST);
  }

  @ExceptionHandler(value = {NoResourceFoundException.class})
  protected ResponseEntity<Object> handleNoResourceFoundException(NoResourceFoundException e) {
    return new ResponseEntity<>(e.getMessage(), null, e.getStatusCode());
  }
}
