package org.cloudfoundry.autoscaler.scheduler.rest;

import java.util.List;

import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.Logger;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationScalingSchedules;
import org.cloudfoundry.autoscaler.scheduler.service.ScheduleManager;
import org.cloudfoundry.autoscaler.scheduler.util.error.InvalidDataException;
import org.cloudfoundry.autoscaler.scheduler.util.error.ValidationErrorResult;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestMethod;
import org.springframework.web.bind.annotation.RestController;

/**
 * 
 *
 */
@RestController
@RequestMapping(value = "/v2/schedules/{app_id}")
public class ScheduleRestController {

	@Autowired
	private ValidationErrorResult validationErrorResult;
	@Autowired
	ScheduleManager scheduleManager;
	private Logger logger = LogManager.getLogger(this.getClass());

	@RequestMapping(method = RequestMethod.GET)
	public ResponseEntity<ApplicationScalingSchedules> getAllSchedules(@PathVariable String app_id) {
		logger.info("Get All schedules for application: " + app_id);
		ApplicationScalingSchedules savedApplicationSchedules = scheduleManager.getAllSchedules(app_id);

		// No schedules found for the specified application return status code NOT_FOUND
		if (savedApplicationSchedules == null) {
			return new ResponseEntity<>(null, null, HttpStatus.NOT_FOUND);
		} else {
			return new ResponseEntity<>(savedApplicationSchedules, null, HttpStatus.OK);
		}

	}

	@RequestMapping(method = RequestMethod.PUT)
	public ResponseEntity<List<String>> createSchedule(@PathVariable String app_id,
			@RequestBody ApplicationScalingSchedules rawApplicationSchedules) {

		validationErrorResult.initEmpty();
		scheduleManager.setUpSchedules(app_id, rawApplicationSchedules);

		logger.info("Validate schedules for application: " + app_id);
		scheduleManager.validateSchedules(app_id, rawApplicationSchedules);

		// If there are no validation errors then proceed with persisting the
		// schedules
		if (!validationErrorResult.hasErrors()) {

			logger.info("Create schedules for application: " + app_id);
			scheduleManager.createSchedules(rawApplicationSchedules);
		}

		if (validationErrorResult.hasErrors()) {
			throw new InvalidDataException();
		}

		return new ResponseEntity<>(null, null, HttpStatus.CREATED);
	}

}
