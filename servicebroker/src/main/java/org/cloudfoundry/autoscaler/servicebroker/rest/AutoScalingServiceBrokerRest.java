package org.cloudfoundry.autoscaler.servicebroker.rest;

import java.util.Locale;

import javax.servlet.http.HttpServletRequest;
import javax.ws.rs.Consumes;
import javax.ws.rs.DELETE;
import javax.ws.rs.GET;
import javax.ws.rs.PUT;
import javax.ws.rs.Path;
import javax.ws.rs.PathParam;
import javax.ws.rs.Produces;
import javax.ws.rs.core.Context;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.common.util.LocaleUtil;
import org.cloudfoundry.autoscaler.common.util.RestApiResponseHandler;
import org.cloudfoundry.autoscaler.servicebroker.Constants;
import org.cloudfoundry.autoscaler.servicebroker.Constants.MESSAGE_KEY;
import org.cloudfoundry.autoscaler.servicebroker.exception.AlreadyBoundAnotherServiceException;
import org.cloudfoundry.autoscaler.servicebroker.exception.ScalingServerFailureException;
import org.cloudfoundry.autoscaler.servicebroker.exception.ServerUrlMappingNotFoundException;
import org.cloudfoundry.autoscaler.servicebroker.exception.ServiceBindingNotFoundException;
import org.cloudfoundry.autoscaler.servicebroker.mgr.ConfigManager;
import org.cloudfoundry.autoscaler.servicebroker.mgr.ScalingServiceMgr;
import org.cloudfoundry.autoscaler.servicebroker.util.MessageUtil;
import org.json.JSONException;
import org.json.JSONObject;

@Path("/v2/")
public class AutoScalingServiceBrokerRest {

	private static final Logger logger = Logger.getLogger(AutoScalingServiceBrokerRest.class);

	@GET
	@Path("catalog")
	@Produces(MediaType.APPLICATION_JSON)
	public Response catalog(@Context final HttpServletRequest httpServletRequest) {

		logger.info(Constants.MSG_ENTRY + " catalog");
		JSONObject catalog = ConfigManager.getCatalogJSON();
		return RestApiResponseHandler.getResponse(Response.Status.OK, catalog.toString());
	}

	@PUT
	@Path("service_instances/{id}")
	@Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response provisionService(@Context final HttpServletRequest httpServletRequest, String jsonRequestStr,
			@PathParam("id") String serviceId) {

		String orgId = null;
		String spaceId = null;

		logger.info("input request:" + jsonRequestStr);

		try {
			JSONObject jsonRequest = new JSONObject(jsonRequestStr);
			orgId = jsonRequest.get("organization_guid").toString();
			spaceId = jsonRequest.get("space_guid").toString();
		} catch (JSONException e) {
			return getResponseError(MESSAGE_KEY.ParseJSONError, e, LocaleUtil.getLocale(httpServletRequest));
		}

		String msgPrefix = String.format("Provisioning service for serviceId %s , orgId %s, spaceId %s ", serviceId,
				orgId, spaceId);
		logger.info(Constants.MSG_ENTRY + msgPrefix);

		try {
			String dashboardUrl = ScalingServiceMgr.getInstance().createService(serviceId, orgId, spaceId);
			JSONObject responseBody = new JSONObject();
			responseBody.put("dashboard_url", dashboardUrl);

			logger.info(Constants.MSG_SUCCESS + msgPrefix + " with response " + responseBody.toString());
			return RestApiResponseHandler.getResponse(Response.Status.CREATED, responseBody);

		} catch (Exception e) {
			logger.error(Constants.MSG_FAIL + msgPrefix + " with execption " + e.getMessage(), e);
			return RestApiResponseHandler.getResponseError(Response.Status.INTERNAL_SERVER_ERROR, e);
		}

	}

	@PUT
	@Path("service_instances/{instance_id}/service_bindings/{id}")
	@Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response bindService(@Context final HttpServletRequest httpServletRequest, String jsonRequestStr,
			@PathParam("instance_id") final String serviceId, @PathParam("id") final String bindingId) {

		String appId = null;
		logger.info("input request:" + jsonRequestStr);
		try {
			JSONObject jsonRequest = new JSONObject(jsonRequestStr);
			appId = jsonRequest.get("app_guid").toString();
		} catch (JSONException e) {
			return getResponseError(MESSAGE_KEY.ParseJSONError, e, LocaleUtil.getLocale(httpServletRequest));
		}

		String msgPrefix = String.format("Bind service for serviceId %s , bindingId %s, appId %s ", serviceId,
				bindingId, appId);
		logger.info(Constants.MSG_ENTRY + msgPrefix);

		try {
			JSONObject credentials = ScalingServiceMgr.getInstance().bindService(appId, serviceId, bindingId);
			logger.info("credentials is " + credentials);
			JSONObject responseBody = new JSONObject();
			responseBody.put("credentials", credentials);
			logger.info(Constants.MSG_SUCCESS + msgPrefix + " with response " + responseBody.toString());
			return Response.ok(responseBody.toString()).build();

		} catch (AlreadyBoundAnotherServiceException e) {
			logger.error(Constants.MSG_FAIL + msgPrefix + " with execption " + e.getMessage(), e);
			return getResponseError(MESSAGE_KEY.AlreadyBindedAnotherService, e, httpServletRequest.getLocale());
		} catch (ServerUrlMappingNotFoundException e) {
			logger.error(Constants.MSG_FAIL + msgPrefix + " with execption " + e.getMessage(), e);
			return getResponseError(MESSAGE_KEY.ServerUrlMappingNotFound, e, httpServletRequest.getLocale());
		} catch (ScalingServerFailureException e) {
			logger.error(Constants.MSG_FAIL + msgPrefix + " with execption " + e.getMessage(), e);
			return getResponseError(MESSAGE_KEY.BindServiceFail, e, httpServletRequest.getLocale());
		} catch (Exception e) {
			logger.error(Constants.MSG_FAIL + msgPrefix + " with execption " + e.getMessage(), e);
			return RestApiResponseHandler.getResponseError(Response.Status.INTERNAL_SERVER_ERROR, e);
		}

	}

	@DELETE
	@Path("service_instances/{instance_id}/service_bindings/{id}")
	@Produces(MediaType.APPLICATION_JSON)
	public Response unbindService(@Context final HttpServletRequest httpServletRequest,
			@PathParam("instance_id") final String serviceId, @PathParam("id") final String bindingId) {

		String msgPrefix = String.format("Unbind service for serviceId %s , bindingId %s", serviceId, bindingId);
		logger.info(Constants.MSG_ENTRY + msgPrefix);

		try {
			ScalingServiceMgr.getInstance().unbindService(serviceId, bindingId);
			logger.info(Constants.MSG_SUCCESS + msgPrefix + " with OK");
			// Note, here need to return {} as empty String is not acceptable
			return RestApiResponseHandler.getResponse(Response.Status.OK, new JSONObject());
		} catch (ServiceBindingNotFoundException e) {
			logger.info(Constants.MSG_SUCCESS + msgPrefix + " with GONE");
			return RestApiResponseHandler.getResponse(Response.Status.GONE, new JSONObject());
		} catch (ScalingServerFailureException e) {
			logger.error(Constants.MSG_FAIL + msgPrefix + " with execption " + e.getMessage(), e);
			return getResponseError(MESSAGE_KEY.UnbindServiceFail, e, LocaleUtil.getLocale(httpServletRequest));
		} catch (Exception e) {
			logger.error(Constants.MSG_FAIL + msgPrefix + " with execption " + e.getMessage(), e);
			return RestApiResponseHandler.getResponseError(Response.Status.INTERNAL_SERVER_ERROR, e);
		}

	}

	@DELETE
	@Path("service_instances/{id}")
	@Produces(MediaType.APPLICATION_JSON)
	public Response deprovisionService(@Context final HttpServletRequest httpServletRequest,
			@PathParam("id") String serviceId) {

		String msgPrefix = String.format("Deprovisioning service for serviceId %s", serviceId);
		logger.info(Constants.MSG_ENTRY + msgPrefix);

		try {
			ScalingServiceMgr.getInstance().deprovisionService(serviceId);
			// Note, here need to return {} as empty String is not acceptable
			logger.info(Constants.MSG_SUCCESS + msgPrefix + " with OK");
			return RestApiResponseHandler.getResponse(Response.Status.OK, new JSONObject());
		} catch (ServiceBindingNotFoundException e) {
			logger.info(Constants.MSG_SUCCESS + msgPrefix + " with GONE");
			return RestApiResponseHandler.getResponse(Response.Status.GONE, new JSONObject());
		} catch (Exception e) {
			logger.error(Constants.MSG_FAIL + msgPrefix + " with execption " + e.getMessage(), e);
			return RestApiResponseHandler.getResponseError(Response.Status.INTERNAL_SERVER_ERROR, e);
		}
	}

	private Response getResponseError(MESSAGE_KEY key, Exception e, Locale locale) {
		String msg = MessageUtil.getMessageString(key.name(), locale);

		return RestApiResponseHandler.getResponseError(msg, key.getErrorCode(), e);
	}

}
