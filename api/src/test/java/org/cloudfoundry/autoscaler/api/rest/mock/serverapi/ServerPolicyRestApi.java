package org.cloudfoundry.autoscaler.api.rest.mock.serverapi;

import static org.cloudfoundry.autoscaler.api.Constants.TESTAPPID;
import static org.cloudfoundry.autoscaler.api.Constants.TESTPOLICYID;

import javax.servlet.http.HttpServletRequest;
import javax.ws.rs.Consumes;
import javax.ws.rs.DELETE;
import javax.ws.rs.GET;
import javax.ws.rs.POST;
import javax.ws.rs.PUT;
import javax.ws.rs.Path;
import javax.ws.rs.PathParam;
import javax.ws.rs.Produces;
import javax.ws.rs.QueryParam;
import javax.ws.rs.core.Context;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

import org.cloudfoundry.autoscaler.api.util.RestApiResponseHandler;
import org.json.JSONObject;

@Path("/resources")
public class ServerPolicyRestApi {

	/**
	 * Creates a new policy
	 * 
	 * @param orgId
	 * @param spaceId
	 * @param jsonString
	 * @return
	 */
	@POST
	@Path("/policies")
	@Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response createPolicy(@Context final HttpServletRequest httpServletRequest, @QueryParam("org") String orgId,
			@QueryParam("space") String spaceId, String jsonString) {

		JSONObject response = new JSONObject();
		response.put("policyId", TESTPOLICYID);
		return RestApiResponseHandler.getResponseCreatedOk(response.toString());

	}

	/**
	 * Updates a policy
	 * 
	 * @param policyId
	 * @param jsonString
	 * @return
	 */
	@PUT
	@Path("/policies/{id}")
	@Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response updatePolicy(@Context final HttpServletRequest httpServletRequest, @PathParam("id") String policyId,
			String jsonString) {
		return RestApiResponseHandler.getResponseOk(new JSONObject());

	}

	@PUT
	@Path("/apps/{id}")
	@Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response attachPolicy(@Context final HttpServletRequest httpServletRequest, @PathParam("id") String appId,
			String jsonString) {
		return RestApiResponseHandler.getResponseOk("{}");
	}

	@DELETE
	@Path("/apps/{id}")
	@Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response detachPolicy(@Context final HttpServletRequest httpServletRequest, @PathParam("id") String appId,
			@QueryParam("policyId") String policyId, @QueryParam("state") String state) {
		return RestApiResponseHandler.getResponseOk("{}");
	}

	/**
	 * Gets a policy
	 * 
	 * @param policyId
	 * @return
	 */
	@GET
	@Path("/policies/{policyId}")
	@Produces(MediaType.APPLICATION_JSON)
	public Response getPolicy(@Context final HttpServletRequest httpServletRequest,
			@PathParam("policyId") String policyId) {
		String jsonStr = "{\"new\":false,\"scheduledPolicies\":{\"59c1793e-5188-443d-bc08-100bfa7e20a8\":{\"minInstCount\":1,\"repeatCycle\":\"[\\\"1\\\",\\\"2\\\",\\\"3\\\",\\\"4\\\",\\\"5\\\",\\\"6\\\",\\\"7\\\"]\",\"maxInstCount\":2,\"timezone\":\"(GMT +08:00) Antarctica\\/Casey\",\"instanceMaxCount\":2,\"startTime\":\"00:00\",\"instanceMinCount\":1,\"endTime\":\"23:59\",\"type\":\"RECURRING\",\"repeatOn\":\"[\\\"1\\\",\\\"2\\\",\\\"3\\\",\\\"4\\\",\\\"5\\\",\\\"6\\\",\\\"7\\\"]\"},\"f06030e0-36a3-4d7a-b013-9e6a750d87cf\":{\"minInstCount\":1,\"repeatCycle\":\"\",\"maxInstCount\":2,\"endDate\":\"2016-04-01\",\"timezone\":\"(GMT +08:00) Antarctica\\/Casey\",\"instanceMaxCount\":2,\"startTime\":\"00:00\",\"instanceMinCount\":1,\"endTime\":\"23:59\",\"type\":\"SPECIALDATE\",\"startDate\":\"2016-04-01\"}},\"attachments\":null,\"specificDate\":[{\"minInstCount\":1,\"repeatCycle\":\"\",\"maxInstCount\":2,\"endDate\":\"2016-04-01\",\"timezone\":\"(GMT +08:00) Antarctica\\/Casey\",\"instanceMaxCount\":2,\"startTime\":\"00:00\",\"instanceMinCount\":1,\"endTime\":\"23:59\",\"type\":\"SPECIALDATE\",\"startDate\":\"2016-04-01\"}],\"policyName\":null,\"policyTriggers\":[{\"upperThreshold\":80,\"instanceStepCountUp\":1,\"statType\":\"average\",\"scaleInAdjustmentType\":\"changeCapacity\",\"scaleInAdjustment\":\"changeCapacity\",\"scaleOutAdjustment\":\"changeCapacity\",\"stepDownCoolDownSecs\":600,\"scaleOutAdjustmentType\":\"changeCapacity\",\"metricType\":\"Memory\",\"statWindow\":300,\"unit\":\"percent\",\"instanceStepCountDown\":-1,\"breachDuration\":600,\"startTime\":0,\"endTime\":0,\"lowerThreshold\":30,\"endSetNumInstances\":10,\"stepUpCoolDownSecs\":600,\"startSetNumInstances\":10}],\"timezone\":\"(GMT +08:00) Antarctica\\/Casey\",\"instanceMinCount\":1,\"type\":\"AutoScalerPolicy\",\"orgId\":\"800faf6a-8413-45a4-b7c7-b9760061bf31\",\"revision\":\"2-e6c41cf81021c692b3170afbc530acac\",\"spaceId\":\"b33cd8c3-455a-4da9-985c-01c2d3f44bd4\",\"policyId\":\"cb032b44-a2e9-454c-87fb-a3fabedcefca\",\"recurringSchedule\":[{\"minInstCount\":1,\"repeatCycle\":\"[\\\"1\\\",\\\"2\\\",\\\"3\\\",\\\"4\\\",\\\"5\\\",\\\"6\\\",\\\"7\\\"]\",\"maxInstCount\":2,\"timezone\":\"(GMT +08:00) Antarctica\\/Casey\",\"instanceMaxCount\":2,\"startTime\":\"00:00\",\"instanceMinCount\":1,\"endTime\":\"23:59\",\"type\":\"RECURRING\",\"repeatOn\":\"[\\\"1\\\",\\\"2\\\",\\\"3\\\",\\\"4\\\",\\\"5\\\",\\\"6\\\",\\\"7\\\"]\"}],\"instanceMaxCount\":5,\"conflicts\":null,\"revisions\":null,\"_id\":\"cb032b44-a2e9-454c-87fb-a3fabedcefca\",\"id\":\"cb032b44-a2e9-454c-87fb-a3fabedcefca\",\"currentScheduledPolicyId\":\"f06030e0-36a3-4d7a-b013-9e6a750d87cf\"}";
		return RestApiResponseHandler.getResponseOk(jsonStr);

	}

	/**
	 * Deletes a policy
	 * 
	 * @param org
	 * @param space
	 * @param id
	 * @return
	 */
	@DELETE
	@Path("/policies/{id}")
	public Response deletePolicy(@Context final HttpServletRequest httpServletRequest, @PathParam("id") String id) {

		return RestApiResponseHandler.getResponse204Ok();
	}

	@GET
	@Path("/history")
	@Produces(MediaType.APPLICATION_JSON)
	public Response getHistoryList(@Context final HttpServletRequest httpServletRequest,
			@QueryParam("appId") String appId, @QueryParam("startTime") String startTime,
			@QueryParam("endTime") String endTime, @QueryParam("status") String status,
			@QueryParam("metrics") String metrics, @QueryParam("offset") String offset,
			@QueryParam("maxCount") String maxCount, @QueryParam("scaleType") String scaleType,
			@QueryParam("timeZone") String timeZone) {
		String historyStr = "[{\"appId\":\"" + TESTAPPID
				+ "\",\"status\":3,\"adjustment\":1,\"instances\":2,\"startTime\":1459491827168,\"endTime\":1459491840041,\"trigger\":{\"metrics\":\"JVMHeapUsage\",\"threshold\":80,\"thresholdType\":\"upper\",\"breachDuration\":60,\"triggerType\":0},\"errorCode\":null,\"scheduleType\":null,\"timeZone\":null,\"scheduleStartTime\":null,\"dayOfWeek\":null,\"rawOffset\":0,\"type\":\"ScalingHistory\",\"id\":\"24bb82a3-6b00-4b44-830b-530ab93b23dd\",\"revision\":\"1-3946e738667603fa87a56044ae298613\",\"new\":false,\"attachments\":null,\"revisions\":null,\"conflicts\":null},{\"appId\":\""
				+ TESTAPPID
				+ "\",\"status\":3,\"adjustment\":1,\"instances\":2,\"startTime\":1458977959975,\"endTime\":1458977978667,\"trigger\":{\"metrics\":\"JVMHeapUsage\",\"threshold\":80,\"thresholdType\":\"upper\",\"breachDuration\":60,\"triggerType\":0},\"errorCode\":null,\"scheduleType\":null,\"timeZone\":null,\"scheduleStartTime\":null,\"dayOfWeek\":null,\"rawOffset\":0,\"type\":\"ScalingHistory\",\"id\":\"211d3ff0-3797-4763-8ef7-86da8816e311\",\"revision\":\"1-c1c81dc58c25ba2f7015ce50569f3649\",\"new\":false,\"attachments\":null,\"revisions\":null,\"conflicts\":null}]";
		return RestApiResponseHandler.getResponseOk(historyStr);
	}
	@GET
    @Path ("/apps/{id}")
    @Produces(MediaType.APPLICATION_JSON)
    public Response getApplication(@Context final HttpServletRequest httpServletRequest, @PathParam("id") String appId, @QueryParam("serviceId") String serviceId, @QueryParam("checkEnablement") String checkEnablement) {
        
			String str = "{\"policyId\":\""+ TESTPOLICYID +"\",\"state\":\"enabled\",\"type\":\"nodejs\",\"config\":{\"reportInterval\":120,\"metricsConfig\":{\"poller\":[\"Memory\"]}}}";
			
			
			return RestApiResponseHandler.getResponseOk(str);
		
    }
//    /**
//     * Get applications that are bond to the service
//     * 
//     * @param serviceId
//     * @return
//     */
//    @GET
//    @Produces(MediaType.APPLICATION_JSON)
//    public Response getApplications(@Context final HttpServletRequest httpServletRequest, @QueryParam("serviceId") String serviceId) {
//        logger.debug("Get applications. Service ID is " + serviceId);
//        ApplicationManager appManager = ApplicationManagerImpl.getInstance();
//        JSONArray jsonArray = new JSONArray();
//        try {
//            List<Application> applications = appManager.getApplications(serviceId);
//            if (applications == null) {
//            	return RestApiResponseHandler.getResponse(Status.CREATED, jsonArray.toString());
//            }
//            //
//            for (Application app : applications) {
//                JSONObject jsonObj = new JSONObject();
//                String appId = app.getAppId();
//                jsonObj.put("appId", app.getAppId());
//                jsonObj.put("appName",  CloudFoundryManager.getInstance().getAppNameByAppId(appId));
//                jsonArray.put(jsonObj);
//            }
//            return RestApiResponseHandler.getResponse(Status.CREATED, jsonArray.toString());
//        } catch (DataStoreException e) {
//            return RestApiResponseHandler.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_database_error, e, httpServletRequest.getLocale());
//        } catch (CloudException e) {
//        	return RestApiResponseHandler.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_cloud_error, e, httpServletRequest.getLocale());
//        } catch (Exception e) {
//            return RestApiResponseHandler.getResponseError(Response.Status.INTERNAL_SERVER_ERROR, e);
//        }
//    }

}
