package org.cloudfoundry.autoscaler.metric.rest;

import java.util.HashMap;

import javax.ws.rs.Consumes;
import javax.ws.rs.DELETE;
import javax.ws.rs.GET;
import javax.ws.rs.PUT;
import javax.ws.rs.Path;
import javax.ws.rs.PathParam;
import javax.ws.rs.Produces;
import javax.ws.rs.core.Application;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;
import javax.ws.rs.core.Response.Status;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.bean.Metric;
import org.cloudfoundry.autoscaler.common.util.CloudFoundryManager;
import org.cloudfoundry.autoscaler.common.util.RestApiResponseHandler;
import org.cloudfoundry.autoscaler.metric.bean.ApplicationMetrics;
import org.cloudfoundry.autoscaler.metric.monitor.MonitorController;
import org.json.JSONObject;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;

@Path("/test")
public class TestModeREST extends Application {
	private static final Logger logger = Logger.getLogger(TestModeREST.class);
	private static final ObjectMapper mapper = new ObjectMapper();

	@GET
	@Path("/{org}/{space}/{appName}")
	@Produces(MediaType.APPLICATION_JSON)
	public Response getAppMetrics(@PathParam("org") String org, @PathParam("space") String space,
			@PathParam("appName") String appName) {

		try {
			MonitorController controller = MonitorController.getInstance();

			String appId = CloudFoundryManager.getInstance().getAppIdByOrgSpaceAppName(org, space, appName);
			logger.debug("appId: " + appId);
			if (appId == null) {
				logger.warn("Can't get the appId for app " + appName);
				return RestApiResponseHandler.getResponseOk(new JSONObject());
			}
			ApplicationMetrics stats = controller.getAppMetrics(appId);
			if (stats == null) {
				logger.warn("Can't get metrics for app " + appName + "(" + appId + ")");
				return RestApiResponseHandler.getResponseOk(new JSONObject());
			}
			return RestApiResponseHandler.getResponseOk(mapper.writeValueAsString(stats));
		} catch (Exception e) {
			logger.error("Internal_Server_Error", e);
			return RestApiResponseHandler.getResponse(Status.INTERNAL_SERVER_ERROR);
		}

	}

	@Path("/metrics/{appId}")
	@PUT
	@Consumes(MediaType.APPLICATION_JSON)
	public Response addMetricTestMode(@PathParam("appId") String appId, String jsonString) {
		try {

			HashMap<String, Metric> testMetrics = mapper.readValue(jsonString,
					new TypeReference<HashMap<String, Metric>>() {
					});
			MonitorController.getInstance().addTestMetrics(appId, testMetrics);
			logger.info("Test Mode turned on for application " + appId);
			return RestApiResponseHandler.getResponseOk();
		} catch (Exception e) {
			logger.error("Internal_Server_Error", e);
			return RestApiResponseHandler.getResponse(Status.INTERNAL_SERVER_ERROR);
		}
	}

	/**
	 * Turn off test mode for appId
	 * 
	 */
	@Path("/metrics/{appId}")
	@DELETE
	@Consumes(MediaType.APPLICATION_JSON)
	public Response removeMetricTestMode(@PathParam("appId") String appId) {
		try {
			MonitorController.getInstance().removeTestMetrics(appId);
			logger.info("Test Mode turned off for application " + appId);
			return RestApiResponseHandler.getResponseOk();
		} catch (Exception e) {
			logger.error("Internal_Server_Error", e);
			return RestApiResponseHandler.getResponse(Status.INTERNAL_SERVER_ERROR);
		}
	}

}
