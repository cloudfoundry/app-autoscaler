package org.cloudfoundry.autoscaler.rest;

import java.io.IOException;
import java.util.ArrayList;

import javax.servlet.http.HttpServletRequest;
import javax.ws.rs.Consumes;
import javax.ws.rs.POST;
import javax.ws.rs.Path;
import javax.ws.rs.core.Context;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.bean.MonitorTriggerEvent;
import org.cloudfoundry.autoscaler.common.util.RestApiResponseHandler;
import org.cloudfoundry.autoscaler.constant.Constants.MESSAGE_KEY;
import org.cloudfoundry.autoscaler.manager.ScalingEventManager;

import com.fasterxml.jackson.databind.ObjectMapper;

@Path("/events")
public class EventRestApi {
	private static final String CLASS_NAME = EventRestApi.class.getName();
	private static final Logger logger = Logger.getLogger(CLASS_NAME);
	private static final ObjectMapper mapper = new ObjectMapper();

	/**
	 * Receives events from monitor service
	 * 
	 * @param jsonString
	 * @return
	 */
	@POST
	@Consumes(MediaType.APPLICATION_JSON)
	public Response receiveEvents(@Context final HttpServletRequest httpServletRequest, String jsonString) {
		logger.info("Received event, JSON string: " + jsonString);
		try {
			MonitorTriggerEvent[] event = mapper.readValue(jsonString, MonitorTriggerEvent[].class);
			ArrayList<MonitorTriggerEvent> eventList = new ArrayList<MonitorTriggerEvent>();
			for (int i = 0; i < event.length; i++) {
				eventList.add(event[i]);
			}
			ScalingEventManager.getInstance().postTriggerEvents(eventList);
			return RestApiResponseHandler.getResponseOk("{}");
		} catch (IOException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_build_JSON_error, e,
					httpServletRequest.getLocale());
		}
	}

}
