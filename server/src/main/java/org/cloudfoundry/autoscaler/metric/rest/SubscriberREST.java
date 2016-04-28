package org.cloudfoundry.autoscaler.metric.rest;

import javax.ws.rs.Consumes;
import javax.ws.rs.DELETE;
import javax.ws.rs.POST;
import javax.ws.rs.Path;
import javax.ws.rs.PathParam;
import javax.ws.rs.core.Application;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;
import javax.ws.rs.core.Response.Status;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.bean.Trigger;
import org.cloudfoundry.autoscaler.common.util.RestApiResponseHandler;
import org.cloudfoundry.autoscaler.exceptions.TriggerNotFoundException;
import org.cloudfoundry.autoscaler.metric.monitor.MonitorController;

import com.fasterxml.jackson.databind.ObjectMapper;

@Path("/triggers")
public class SubscriberREST extends Application {
	private static final Logger logger = Logger.getLogger(SubscriberREST.class);
	private static final ObjectMapper mapper = new ObjectMapper();

	@POST
	@Consumes(MediaType.APPLICATION_JSON)
	public Response addTrigger(String jsonString) {
		try {
			Trigger trigger = mapper.readValue(jsonString, Trigger.class);
			if (!"".equals(trigger.getAppId())) {
				MonitorController.getInstance().addTrigger(trigger);
			}
			return RestApiResponseHandler.getResponseOk("Trigger added");

		} catch (Exception e) {
			logger.error(e.getMessage(), e);
			return RestApiResponseHandler.getResponse(Status.INTERNAL_SERVER_ERROR);
		}
	}

	@DELETE
	@Path("/{appId}")
	@Consumes(MediaType.APPLICATION_JSON)
	public Response removeTrigger(@PathParam("appId") String appId) {
		try {
			MonitorController.getInstance().removeTrigger(appId);
		} catch (TriggerNotFoundException e) {
			return RestApiResponseHandler.getResponse(Status.NOT_FOUND, TriggerNotFoundException.ERROR_CODE);
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
			return RestApiResponseHandler.getResponseError(Status.INTERNAL_SERVER_ERROR, e);
		}

		return RestApiResponseHandler.getResponseOk("Trigger removed");
	}
}
