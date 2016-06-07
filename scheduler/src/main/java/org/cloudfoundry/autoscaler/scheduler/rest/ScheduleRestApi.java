package org.cloudfoundry.autoscaler.scheduler.rest;

import org.apache.commons.logging.Log;
import org.apache.commons.logging.LogFactory;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationScalingSchedules;
import org.cloudfoundry.autoscaler.scheduler.service.ScalingScheduleManager;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestMethod;
import org.springframework.web.bind.annotation.RestController;

/**
 * @author Fujitsu
 *
 */
@RestController
@RequestMapping(value = "/v2/schedules")
public class ScheduleRestApi {

	@Autowired
	ScalingScheduleManager scalingScheduleManager;

	private Log logger = LogFactory.getLog(this.getClass());

	@RequestMapping(method = RequestMethod.PUT)
	public ResponseEntity<ApplicationScalingSchedules> createSchedule(
			@RequestBody ApplicationScalingSchedules rawApplicationSchedules) {

		logger.info("create schedule");
		String appId = scalingScheduleManager.createSchedules(rawApplicationSchedules);
		ApplicationScalingSchedules savedApplicationSchedules = scalingScheduleManager.getSchedules(appId);

		return new ResponseEntity<>(savedApplicationSchedules, null, HttpStatus.CREATED);
	}

	// @RequestMapping(method = RequestMethod.GET,value="/app/{id}")
	// @RequestMapping(method = RequestMethod.DELETE,value="/app/{id}")

}
