package org.cloudfoundry.autoscaler.scheduler.rest;

import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.Parameter;
import io.swagger.v3.oas.annotations.media.Content;
import io.swagger.v3.oas.annotations.media.Schema;
import io.swagger.v3.oas.annotations.responses.ApiResponse;
import io.swagger.v3.oas.annotations.responses.ApiResponses;
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
  @Operation(
      summary = "Get all schedules (specific dates and recurring) for the specified application id.")
  @ApiResponses(
      value = {
          @ApiResponse(responseCode = "200", description = "Schedules found for the specified application id.", content = @Content(mediaType = "application/json", schema = @Schema(implementation = ApplicationSchedules.class))),
        @ApiResponse(responseCode = "404", description = "No schedules found for the specified application id.")
      })
  public ResponseEntity<ApplicationSchedules> getAllSchedules(
      @Parameter(name = "app_id", description = "The application id", required = true)
          @PathVariable("app_id")
          @NotNull
          @UUID
          String appId) {
    logger.info("Get All schedules for application: {}", appId);

    ApplicationSchedules savedApplicationSchedules = scheduleManager.getAllSchedules(appId);

    // No schedules found for the specified application return status code NOT_FOUND
    if (!savedApplicationSchedules.getSchedules().hasSchedules()) {
      return new ResponseEntity<>(HttpStatus.NOT_FOUND);
    } else {
      return new ResponseEntity<>(savedApplicationSchedules, HttpStatus.OK);
    }
  }

  @PutMapping
  @ResponseStatus(HttpStatus.OK)
  @Operation(
      summary = "Create/Modify schedules for the specified application id.")
  @ApiResponses(
      value = {
        @ApiResponse(responseCode = "200", description = "Schedules created for the specified application id."),
        @ApiResponse(responseCode = "204", description = "Schedules modified for the specified application id.")
      })
  public ResponseEntity<List<String>> createSchedules(
      @Parameter(name = "app_id", description = "The application id", required = true)
          @PathVariable("app_id")
          @NotNull
          @UUID
          String appId,
      @Parameter(name = "guid", description = "The policy guid", required = true)
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
      return new ResponseEntity<>(HttpStatus.NO_CONTENT);
    }

    return new ResponseEntity<>(HttpStatus.OK);
  }

  @DeleteMapping
  @ResponseStatus(HttpStatus.NO_CONTENT)
  @Operation(
      summary =
          "Delete all schedules (specific dates and recurring) for the specified application id.")
  @ApiResponses(
      value = {
        @ApiResponse(
            responseCode = "204",
            description = "All schedules deleted for the specified application id."),
        @ApiResponse(
            responseCode = "404",
            description = "No schedules found for deletion for the specified application id.")
      })
  public ResponseEntity<List<String>> deleteSchedules(
      @Parameter(name = "app_id", description = "The application id", required = true)
          @PathVariable("app_id")
          @NotNull
          @UUID
          String appId) {

    Schedules existingSchedules = scheduleManager.getAllSchedules(appId).getSchedules();
    if (!existingSchedules.hasSchedules()) {
      return new ResponseEntity<>(HttpStatus.NOT_FOUND);
    }

    logger.info("Delete schedules for application: {}", appId);
    scheduleManager.deleteSchedules(appId);

    return new ResponseEntity<>(HttpStatus.NO_CONTENT);
  }
}
