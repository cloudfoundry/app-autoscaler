package org.cloudfoundry.autoscaler.rest;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

import javax.servlet.http.HttpServletRequest;
import javax.ws.rs.Consumes;
import javax.ws.rs.DELETE;
import javax.ws.rs.GET;
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
import org.cloudfoundry.autoscaler.common.util.CloudFoundryManager;
import org.cloudfoundry.autoscaler.common.util.RestApiResponseHandler;
import org.cloudfoundry.autoscaler.constant.Constants.MESSAGE_KEY;
import org.cloudfoundry.autoscaler.data.couchdb.document.Application;
import org.cloudfoundry.autoscaler.exceptions.CloudException;
import org.cloudfoundry.autoscaler.exceptions.DataStoreException;
import org.cloudfoundry.autoscaler.exceptions.MetricNotSupportedException;
import org.cloudfoundry.autoscaler.exceptions.MonitorServiceException;
import org.cloudfoundry.autoscaler.exceptions.PolicyNotFoundException;
import org.cloudfoundry.autoscaler.manager.ApplicationManager;
import org.cloudfoundry.autoscaler.manager.ApplicationManagerImpl;
import org.cloudfoundry.autoscaler.util.MetricConfigManager;
import org.json.JSONArray;
import org.json.JSONObject;

import com.fasterxml.jackson.databind.ObjectMapper;

@Path("/apps")
public class ApplicationRestApi {
	private static final String CLASS_NAME = ApplicationRestApi.class.getName();
	private static final Logger logger = Logger.getLogger(CLASS_NAME);
	private static final ObjectMapper objectMapper = new ObjectMapper();

	/*
	 * Sample request JSON: {"policyId": "xxx"}
	 */
	@PUT
	@Path("/{id}")
	@Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response attachPolicy(@Context final HttpServletRequest httpServletRequest, @PathParam("id") String appId,
			String jsonString) throws MonitorServiceException {
		logger.info("Attach policy for application " + appId + ". Received JSON string: " + jsonString);
		ApplicationManager appManager = ApplicationManagerImpl.getInstance();
		try {
			JSONObject jsonObj = new JSONObject(jsonString);
			String policyId = (String) jsonObj.get("policyId");
			String state = (String) jsonObj.get("state");
			appManager.attachPolicy(appId, policyId, state);
		} catch (DataStoreException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_database_error, e,
					httpServletRequest.getLocale());
		} catch (MetricNotSupportedException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_metric_not_supported_error, e,
					httpServletRequest.getLocale());
		} catch (PolicyNotFoundException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_policy_not_found_error, e,
					httpServletRequest.getLocale());
		} catch (Exception e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_parse_JSON_error, e,
					httpServletRequest.getLocale());
		}
		return RestApiResponseHandler.getResponseOk("{}");
	}

	/*
	 * Sample request JSON: {"policyId": "xxx"}
	 */
	@DELETE
	@Path("/{id}")
	@Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response detachPolicy(@Context final HttpServletRequest httpServletRequest, @PathParam("id") String appId,
			@QueryParam("policyId") String policyId, @QueryParam("state") String state) {
		logger.info("Detach policy for application " + appId + ". policyId: " + policyId + " state: " + state);
		ApplicationManager appManager = ApplicationManagerImpl.getInstance();
		try {
			appManager.detachPolicy(appId, policyId, state);
		} catch (DataStoreException e) {

			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_database_error, e,
					httpServletRequest.getLocale());
		} catch (MonitorServiceException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_cloud_error, e,
					httpServletRequest.getLocale());
		} catch (MetricNotSupportedException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_metric_not_supported_error, e,
					httpServletRequest.getLocale());
		} catch (PolicyNotFoundException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_policy_not_found_error, e,
					httpServletRequest.getLocale());
		} catch (Exception e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_parse_JSON_error, e,
					httpServletRequest.getLocale());
		}
		return RestApiResponseHandler.getResponseOk("{}");
	}

	/**
	 * Gets policy of the application
	 * 
	 * @param appId
	 * @return
	 */
	@GET
	@Path("{id}")
	@Produces(MediaType.APPLICATION_JSON)
	public Response getApplication(@Context final HttpServletRequest httpServletRequest, @PathParam("id") String appId,
			@QueryParam("serviceId") String serviceId, @QueryParam("checkEnablement") String checkEnablement) {
		try {

			ApplicationManager appManager = ApplicationManagerImpl.getInstance();
			Application app = appManager.getApplication(appId);

			Map<String, Object> json = new HashMap<String, Object>();
			String appType = null;
			if (app.getAppType() != null)
				appType = app.getAppType();
			if (appType != null)
				json.put("type", appType);
			json.put("policyId", app.getPolicyId());
			json.put("state", app.getPolicyState());

			MetricConfigManager configService = MetricConfigManager.getInstance();
			Map<String, Object> configMap = configService.loadDefaultConfig(appType, appId);
			json.put("config", configMap);
			return RestApiResponseHandler.getResponseOk(objectMapper.writeValueAsString(json));
		} catch (DataStoreException e) {

			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_database_error, e,
					httpServletRequest.getLocale());
		} catch (CloudException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_cloud_error, e,
					httpServletRequest.getLocale());
		} catch (Exception e) {
			logger.error(e.getMessage(), e);
			return Response.status(Status.INTERNAL_SERVER_ERROR).build();
		}
	}

	/**
	 * Get applications that are bond to the service
	 * 
	 * @param serviceId
	 * @return
	 */
	@GET
	@Produces(MediaType.APPLICATION_JSON)
	public Response getApplications(@Context final HttpServletRequest httpServletRequest,
			@QueryParam("serviceId") String serviceId) {
		logger.debug("Get applications. Service ID is " + serviceId);
		ApplicationManager appManager = ApplicationManagerImpl.getInstance();
		JSONArray jsonArray = new JSONArray();
		try {
			List<Application> applications = appManager.getApplications(serviceId);
			if (applications == null) {
				return RestApiResponseHandler.getResponse(Status.CREATED, jsonArray.toString());
			}
			//
			for (Application app : applications) {
				JSONObject jsonObj = new JSONObject();
				String appId = app.getAppId();
				jsonObj.put("appId", app.getAppId());
				jsonObj.put("appName", CloudFoundryManager.getInstance().getAppNameByAppId(appId));
				jsonArray.put(jsonObj);
			}
			return RestApiResponseHandler.getResponse(Status.CREATED, jsonArray.toString());
		} catch (DataStoreException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_database_error, e,
					httpServletRequest.getLocale());
		} catch (CloudException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_cloud_error, e,
					httpServletRequest.getLocale());
		} catch (Exception e) {
			return RestApiResponseHandler.getResponseError(Response.Status.INTERNAL_SERVER_ERROR, e);
		}
	}
}
