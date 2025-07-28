package org.cloudfoundry.autoscaler.scheduler.rest;

import io.swagger.annotations.ApiOperation;
import io.swagger.annotations.ApiParam;
import io.swagger.annotations.ApiResponse;
import io.swagger.annotations.ApiResponses;
import jakarta.validation.Valid;
import jakarta.validation.constraints.NotNull;
import java.util.List;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationSchedules;
import org.cloudfoundry.autoscaler.scheduler.rest.model.Schedules;
import org.cloudfoundry.autoscaler.scheduler.service.ScheduleManager;
import org.hibernate.validator.constraints.UUID;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.validation.annotation.Validated;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

/** Controller class for handling the REST api calls. */
@RestController
@RequestMapping(value = "/v1/apps/{app_id}/schedules")
@Validated
public class ScheduleRestController {

  @Autowired ScheduleManager scheduleManager;
  private final Logger logger = LoggerFactory.getLogger(this.getClass());

  @GetMapping
  @ApiOperation(
      value = "Get all schedules (specific dates and recurring) for the specified application id.",
      produces = "application/json")
  @ApiResponses(
      value = {
        @ApiResponse(
            code = 200,
            message = "Schedules found for the specified application id.",
            response = ApplicationSchedules.class),
        @ApiResponse(code = 404, message = "No schedules found for the specified application id.")
      })
  public ResponseEntity<ApplicationSchedules> getAllSchedules(
      @ApiParam(name = "app_id", value = "The application id", required = true)
          @PathVariable("app_id")
          @NotNull
          @UUID
          String appId) {
    logger.info("Get All schedules for application: {}", appId);

    ApplicationSchedules savedApplicationSchedules = scheduleManager.getAllSchedules(appId);

    // No schedules found for the specified application return status code NOT_FOUND
    if (!savedApplicationSchedules.getSchedules().hasSchedules()) {
      return new ResponseEntity<>(null, null, HttpStatus.NOT_FOUND);
    } else {
      return new ResponseEntity<>(savedApplicationSchedules, null, HttpStatus.OK);
    }
  }

  @PutMapping
  @ResponseStatus(HttpStatus.OK)
  @ApiOperation(
      value = "Create/Modify schedules for the specified application id.",
      consumes = "application/json")
  @ApiResponses(
      value = {
        @ApiResponse(code = 200, message = "Schedules created for the specified application id."),
        @ApiResponse(code = 204, message = "Schedules modified for the specified application id.")
      })
  public ResponseEntity<List<String>> createSchedules(
      @ApiParam(name = "app_id", value = "The application id", required = true)
          @PathVariable("app_id")
          @NotNull
          @UUID
          String appId,
      @ApiParam(name = "guid", value = "The policy guid", required = true)
          @RequestParam("guid")
          @NotNull
          @UUID
          String guid,
      @RequestBody @Valid ApplicationSchedules rawApplicationPolicy) {
    // Note: Request could be to update existing schedules or create new schedules.

    scheduleManager.setUpSchedules(appId, guid, rawApplicationPolicy);

    Schedules existingSchedules = scheduleManager.getAllSchedules(appId).getSchedules();
    boolean isUpdateScheduleRequest = existingSchedules.hasSchedules();

    if (isUpdateScheduleRequest) { // Request to update the schedules
      logger.info("Update schedules for application: {}", appId);

      logger.info("Delete existing schedules for application: {}", appId);
      scheduleManager.deleteSchedules(appId);
    }

    if (rawApplicationPolicy.getSchedules() != null) {
      logger.info("Create schedules for application: {}", appId);
      scheduleManager.createSchedules(rawApplicationPolicy.getSchedules());
    }

    if (isUpdateScheduleRequest) {
      return new ResponseEntity<>(null, null, HttpStatus.NO_CONTENT);
    }

    return new ResponseEntity<>(null, null, HttpStatus.OK);
  }

  @DeleteMapping
  @ResponseStatus(HttpStatus.NO_CONTENT)
  @ApiOperation(
      value =
          "Delete all schedules (specific dates and recurring) for the specified application id.")
  @ApiResponses(
      value = {
        @ApiResponse(
            code = 204,
            message = "All schedules deleted for the specified application id."),
        @ApiResponse(
            code = 404,
            message = "No schedules found for deletion for the specified application id.")
      })
  public ResponseEntity<List<String>> deleteSchedules(
      @ApiParam(name = "app_id", value = "The application id", required = true)
          @PathVariable("app_id")
          @NotNull
          @UUID
          String appId) {

    Schedules existingSchedules = scheduleManager.getAllSchedules(appId).getSchedules();
    if (!existingSchedules.hasSchedules()) {
      return new ResponseEntity<>(null, null, HttpStatus.NOT_FOUND);
    }

    logger.info("Delete schedules for application: {}", appId);
    scheduleManager.deleteSchedules(appId);

    return new ResponseEntity<>(null, null, HttpStatus.NO_CONTENT);
  }
}
