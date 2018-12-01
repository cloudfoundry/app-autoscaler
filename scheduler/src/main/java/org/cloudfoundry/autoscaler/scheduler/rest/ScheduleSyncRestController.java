package org.cloudfoundry.autoscaler.scheduler.rest;

import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.Logger;
import org.cloudfoundry.autoscaler.scheduler.rest.model.SynchronizeResult;
import org.cloudfoundry.autoscaler.scheduler.service.ScheduleManager;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestMethod;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping(value = "/v1/syncSchedules")
public class ScheduleSyncRestController {

	@Autowired
	private ScheduleManager scheduleManager;
	private Logger logger = LogManager.getLogger(this.getClass());
	
	@RequestMapping(method = RequestMethod.PUT)
	@ResponseStatus(HttpStatus.OK)
	public ResponseEntity<SynchronizeResult> synchronizeSchedules() {
		logger.info("synchronize schedules");
		SynchronizeResult result = scheduleManager.synchronizeSchedules();
		return new ResponseEntity<>(result, null, HttpStatus.OK);

	}

}
