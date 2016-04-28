package org.cloudfoundry.autoscaler.rest;

import java.io.IOException;
import java.util.HashMap;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.UUID;

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
import javax.ws.rs.core.Response.Status;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.bean.AutoScalerPolicyTrigger;
import org.cloudfoundry.autoscaler.common.util.RestApiResponseHandler;
import org.cloudfoundry.autoscaler.constant.Constants.MESSAGE_KEY;
import org.cloudfoundry.autoscaler.data.couchdb.document.Application;
import org.cloudfoundry.autoscaler.data.couchdb.document.AutoScalerPolicy;
import org.cloudfoundry.autoscaler.data.couchdb.document.ScheduledPolicy;
import org.cloudfoundry.autoscaler.exceptions.DataStoreException;
import org.cloudfoundry.autoscaler.exceptions.MetricNotSupportedException;
import org.cloudfoundry.autoscaler.exceptions.MonitorServiceException;
import org.cloudfoundry.autoscaler.exceptions.NoMonitorServiceBoundException;
import org.cloudfoundry.autoscaler.exceptions.PolicyNotFoundException;
import org.cloudfoundry.autoscaler.manager.ApplicationManager;
import org.cloudfoundry.autoscaler.manager.ApplicationManagerImpl;
import org.cloudfoundry.autoscaler.manager.PolicyManager;
import org.cloudfoundry.autoscaler.manager.PolicyManagerImpl;
import org.cloudfoundry.autoscaler.util.PolicyParser;
import org.json.JSONArray;
import org.json.JSONObject;

import com.fasterxml.jackson.databind.ObjectMapper;

@Path("/policies")
public class PolicyRestApi {
	private static final String CLASS_NAME = PolicyRestApi.class.getName();
	private static final Logger logger = Logger.getLogger(CLASS_NAME);
	private ObjectMapper objectMapper = new ObjectMapper();

	/**
	 * Creates a new policy
	 * 
	 * @param orgId
	 * @param spaceId
	 * @param jsonString
	 * @return
	 */
	@POST
	@Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response createPolicy(@Context final HttpServletRequest httpServletRequest, @QueryParam("org") String orgId,
			@QueryParam("space") String spaceId, String jsonString) {
		try {
			logger.info("Received JSON String of policy content: " + jsonString);
			PolicyManager policyManager = PolicyManagerImpl.getInstance();
			AutoScalerPolicy policy = PolicyParser.parse(jsonString);
			policy.setOrgId(orgId);
			policy.setSpaceId(spaceId);
			generatePolicyFromJsonString(policy, jsonString);
			String policyId = policyManager.createPolicy(policy);
			logger.info("Generate policy with id: " + policyId + " for policy content : " + jsonString);
			JSONObject response = new JSONObject();
			response.put("policyId", policyId);
			return RestApiResponseHandler.getResponse(Status.CREATED, response);
		} catch (IOException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_parse_JSON_error, e,
					httpServletRequest.getLocale());
		} catch (DataStoreException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_database_error, e,
					httpServletRequest.getLocale());
		}
	}

	/**
	 * Updates a policy
	 * 
	 * @param policyId
	 * @param jsonString
	 * @return
	 */
	@PUT
	@Path("/{id}")
	@Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response updatePolicy(@Context final HttpServletRequest httpServletRequest, @PathParam("id") String policyId,
			String jsonString) {
		try {
			logger.debug("Update policy " + policyId + "Received JSON string:\n" + jsonString);
			AutoScalerPolicy policy = PolicyParser.parse(jsonString);
			policy.setPolicyId(policyId);
			generatePolicyFromJsonString(policy, jsonString);
			PolicyManager policyManager = PolicyManagerImpl.getInstance();
			policyManager.updatePolicy(policy);
			ApplicationManager appManager = ApplicationManagerImpl.getInstance();
			List<Application> apps = appManager.getApplicationByPolicyId(policyId);
			for (Application app : apps) {
				appManager.updatePolicyOfApplication(app.getAppId(), app.getPolicyState(), policy);
			}
			policyManager.updatePolicy(policy);
			logger.info("Update policy for id: " + policyId + " for policy content : " + jsonString);
			return RestApiResponseHandler.getResponseOk(new JSONObject());
		} catch (IOException e) {

			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_parse_JSON_error, e,
					httpServletRequest.getLocale());
		} catch (DataStoreException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_database_error, e,
					httpServletRequest.getLocale());
		} catch (MonitorServiceException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_cloud_error, e,
					httpServletRequest.getLocale());
		} catch (MetricNotSupportedException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_metric_not_supported_error, e,
					httpServletRequest.getLocale());
		} catch (NoMonitorServiceBoundException e) {
			return RestApiResponseHandler.getResponseOk(new JSONObject());
		}
	}

	/**
	 * Gets a policy
	 * 
	 * @param policyId
	 * @return
	 */
	@GET
	@Path("/{id}")
	@Produces(MediaType.APPLICATION_JSON)
	public Response getPolicy(@Context final HttpServletRequest httpServletRequest, @PathParam("id") String policyId) {
		if (policyId == null || policyId.trim().length() == 0) {
			return Response.serverError().entity("Policy ID is not specified").build();
		}

		try {
			PolicyManager policyManager = PolicyManagerImpl.getInstance();
			AutoScalerPolicy policy = policyManager.getPolicyById(policyId);
			String jsonString = objectMapper.writeValueAsString(policy);
			JSONObject json = new JSONObject(jsonString);
			JSONObject scheduledPoliciesJson = (JSONObject) json.get("scheduledPolicies");
			JSONArray recurringScheduleJsonArray = new JSONArray();
			JSONArray specificDateJsonArray = new JSONArray();

			Iterator<String> iter = scheduledPoliciesJson.keySet().iterator();
			while (iter.hasNext()) {
				String key = iter.next();
				JSONObject scheduledPolicyItemJson = (JSONObject) scheduledPoliciesJson.get(key);

				if (scheduledPolicyItemJson.get("type").equals(ScheduledPolicy.ScheduledType.RECURRING.name())) {
					scheduledPolicyItemJson.put("minInstCount", scheduledPolicyItemJson.get("instanceMinCount"));
					scheduledPolicyItemJson.put("maxInstCount", scheduledPolicyItemJson.get("instanceMaxCount"));
					scheduledPolicyItemJson.put("repeatOn", scheduledPolicyItemJson.get("repeatCycle"));
					recurringScheduleJsonArray.put(scheduledPolicyItemJson);
				} else if (scheduledPolicyItemJson.get("type")
						.equals(ScheduledPolicy.ScheduledType.SPECIALDATE.name())) {
					scheduledPolicyItemJson.put("minInstCount", scheduledPolicyItemJson.get("instanceMinCount"));
					scheduledPolicyItemJson.put("maxInstCount", scheduledPolicyItemJson.get("instanceMaxCount"));
					String startDateTime = String.valueOf(scheduledPolicyItemJson.get("startTime"));
					String endDateTime = String.valueOf(scheduledPolicyItemJson.get("endTime"));
					scheduledPolicyItemJson.put("startDate", startDateTime.split(" ")[0]);
					scheduledPolicyItemJson.put("startTime", startDateTime.split(" ")[1]);
					scheduledPolicyItemJson.put("endDate", endDateTime.split(" ")[0]);
					scheduledPolicyItemJson.put("endTime", endDateTime.split(" ")[1]);
					specificDateJsonArray.put(scheduledPolicyItemJson);
				}
			}
			json.put("recurringSchedule", recurringScheduleJsonArray);
			json.put("specificDate", specificDateJsonArray);
			return RestApiResponseHandler.getResponseOk(json);
		} catch (IOException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_build_JSON_error, e,
					httpServletRequest.getLocale());
		} catch (PolicyNotFoundException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_policy_not_found_error, e,
					httpServletRequest.getLocale());
		} catch (DataStoreException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_database_error, e,
					httpServletRequest.getLocale());
		}
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
	@Path("/{id}")
	public Response deletePolicy(@Context final HttpServletRequest httpServletRequest, @PathParam("id") String id) {
		try {
			PolicyManager policyManager = PolicyManagerImpl.getInstance();
			policyManager.deletePolicy(id);
			return RestApiResponseHandler.getResponse(Status.NO_CONTENT);
		} catch (PolicyNotFoundException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_policy_not_found_error, e,
					httpServletRequest.getLocale());
		} catch (DataStoreException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_database_error, e,
					httpServletRequest.getLocale());
		}
	}

	/**
	 * Gets brief of policies under the specified service
	 * 
	 * @param serviceId
	 * @return
	 */
	@GET
	@Produces(MediaType.APPLICATION_JSON)
	public Response getPolicies(@Context final HttpServletRequest httpServletRequest,
			@QueryParam("service_id") String serviceId) {
		if (serviceId == null || serviceId.trim().length() == 0) {
			return Response.serverError().entity("Service ID is not specified").build();
		}

		// Get application list that bond to this service
		ApplicationManager appManager = ApplicationManagerImpl.getInstance();
		JSONArray jsonArray = new JSONArray();
		String appId = "";
		String policyId = "";
		String policyState = "";
		String policy_minInst = "";
		String policy_maxInst = "";
		String policy_metrics = "";
		List<Application> applications;
		try {
			applications = appManager.getApplications(serviceId);
			if (applications == null) {
				return RestApiResponseHandler.getResponseOk("{}");
			} else {
				for (Application app : applications) {
					appId = "";
					policyId = "";
					policyState = "";
					policy_minInst = "";
					policy_maxInst = "";
					policy_metrics = "";

					appId = app.getAppId();
					policyId = app.getPolicyId();

					if ((policyId != null) && (policyId.trim().length() > 0)) {
						policyState = app.getPolicyState();

						try {
							PolicyManager policyManager = PolicyManagerImpl.getInstance();
							AutoScalerPolicy policy = policyManager.getPolicyById(policyId);

							policy_minInst = String.valueOf(policy.getInstanceMinCount());
							String currentPolicyId = policy.getCurrentScheduledPolicyId();
							if (null != currentPolicyId) {
								Map<String, ScheduledPolicy> scheduleMap = policy.getScheduledPolicies();
								if (null != scheduleMap) {
									ScheduledPolicy schedule = scheduleMap.get(currentPolicyId);
									if (null != schedule && schedule.getInstanceMinCount() != 0) {
										policy_minInst = String.valueOf(schedule.getInstanceMinCount());
									}
								}
							}
							policy_maxInst = String.valueOf(policy.getInstanceMaxCount());

							List<AutoScalerPolicyTrigger> triggers = policy.getPolicyTriggers();
							StringBuffer buf = new StringBuffer();
							for (AutoScalerPolicyTrigger trigger : triggers) {
								buf.append(trigger.getMetricType() + ";");
							}
							policy_metrics = buf.toString();

						} catch (Exception e) {
							policyState = "--";
							policy_minInst = "--";
							policy_maxInst = "--";
							policy_metrics = "--";
						}
					} else {
						policyState = "--";
						policy_minInst = "--";
						policy_maxInst = "--";
						policy_metrics = "--";
					}

					JSONObject jsonAppPolicy = new JSONObject();
					jsonAppPolicy.put("appId", appId);
					jsonAppPolicy.put("policyId", policyId);
					jsonAppPolicy.put("policyState", policyState);
					jsonAppPolicy.put("policy_minInst", policy_minInst);
					jsonAppPolicy.put("policy_maxInst", policy_maxInst);
					jsonAppPolicy.put("policy_metrics", policy_metrics);

					jsonArray.put(jsonAppPolicy);

				} // end of "for (Application app : applications)"

				return RestApiResponseHandler.getResponseOk(jsonArray.toString());
			}
		} catch (DataStoreException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_database_error, e,
					httpServletRequest.getLocale());
		}

	}

	private void generatePolicyFromJsonString(AutoScalerPolicy policy, String jsonString) throws IOException {
		JSONArray recurringSchedule = null;
		JSONArray specificDate = null;
		JSONObject policy_json = new JSONObject(jsonString);
		if (!(policy_json.isNull("recurringSchedule")))
			recurringSchedule = (JSONArray) policy_json.get("recurringSchedule");
		if (!(policy_json.isNull("specificDate")))
			specificDate = (JSONArray) policy_json.get("specificDate");
		String timezone = String.valueOf(new JSONObject(jsonString).get("timezone"));
		Map<String, ScheduledPolicy> scheduledPolicyMap = new HashMap<String, ScheduledPolicy>();
		if (null != recurringSchedule) {
			Iterator<Object> it = recurringSchedule.iterator();
			while (it.hasNext()) {
				JSONObject recurringScheduleItemJson = (JSONObject) it.next();
				ScheduledPolicy scheduledPolicy = new ScheduledPolicy();
				scheduledPolicy.setType(ScheduledPolicy.ScheduledType.RECURRING.name());
				scheduledPolicy.setInstanceMinCount(
						Integer.parseInt(String.valueOf(recurringScheduleItemJson.get("minInstCount"))));
				scheduledPolicy.setInstanceMaxCount(
						Integer.parseInt(String.valueOf(recurringScheduleItemJson.get("maxInstCount"))));
				scheduledPolicy.setStartTime(String.valueOf(recurringScheduleItemJson.get("startTime")));
				scheduledPolicy.setEndTime(String.valueOf(recurringScheduleItemJson.get("endTime")));
				scheduledPolicy.setRepeatCycle(String.valueOf(recurringScheduleItemJson.get("repeatOn")));
				scheduledPolicy.setTimezone(timezone);
				scheduledPolicyMap.put(UUID.randomUUID().toString(), scheduledPolicy);
			}
			;
		}
		if (null != specificDate) {
			Iterator<Object> it = specificDate.iterator();
			while (it.hasNext()) {
				JSONObject specificDateItemJson = (JSONObject) it.next();
				ScheduledPolicy scheduledPolicy = new ScheduledPolicy();
				scheduledPolicy.setType(ScheduledPolicy.ScheduledType.SPECIALDATE.name());
				scheduledPolicy.setInstanceMinCount(
						Integer.parseInt(String.valueOf(specificDateItemJson.get("minInstCount"))));
				scheduledPolicy.setInstanceMaxCount(
						Integer.parseInt(String.valueOf(specificDateItemJson.get("maxInstCount"))));
				scheduledPolicy.setStartTime(String.valueOf(specificDateItemJson.get("startDate")) + " "
						+ String.valueOf(specificDateItemJson.get("startTime")));
				scheduledPolicy.setEndTime(String.valueOf(specificDateItemJson.get("endDate")) + " "
						+ String.valueOf(specificDateItemJson.get("endTime")));
				scheduledPolicy.setTimezone(timezone);
				scheduledPolicy.setRepeatCycle("");
				scheduledPolicyMap.put(UUID.randomUUID().toString(), scheduledPolicy);
			}
			;
		}

		policy.setScheduledPolicies(scheduledPolicyMap);
	}
}
