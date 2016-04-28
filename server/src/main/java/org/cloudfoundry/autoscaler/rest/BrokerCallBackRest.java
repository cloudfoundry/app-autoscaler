package org.cloudfoundry.autoscaler.rest;

import java.io.IOException;

import javax.servlet.http.HttpServletRequest;
import javax.ws.rs.Consumes;
import javax.ws.rs.DELETE;
import javax.ws.rs.POST;
import javax.ws.rs.Path;
import javax.ws.rs.QueryParam;
import javax.ws.rs.core.Context;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;
import javax.ws.rs.core.Response.Status;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.common.util.RestApiResponseHandler;
import org.cloudfoundry.autoscaler.constant.Constants.MESSAGE_KEY;
import org.cloudfoundry.autoscaler.exceptions.DataStoreException;
import org.cloudfoundry.autoscaler.exceptions.MetricNotSupportedException;
import org.cloudfoundry.autoscaler.exceptions.MonitorServiceException;
import org.cloudfoundry.autoscaler.exceptions.NoAttachedPolicyException;
import org.cloudfoundry.autoscaler.exceptions.PolicyNotFoundException;
import org.cloudfoundry.autoscaler.manager.ApplicationManager;
import org.cloudfoundry.autoscaler.manager.ApplicationManagerImpl;
import org.cloudfoundry.autoscaler.manager.NewAppRequestEntity;
import org.cloudfoundry.autoscaler.metric.monitor.MonitorController;

import com.fasterxml.jackson.databind.ObjectMapper;

@Path("/brokercallback")
public class BrokerCallBackRest {

	private static final String CLASS_NAME = BrokerCallBackRest.class.getName();
	private static final Logger logger = Logger.getLogger(CLASS_NAME);
	private static final ObjectMapper mapper = new ObjectMapper();

	/*
	 * Sample request JSON: {"appId": "xxxx", "serviceId": "xxxx", "bindingId": "xxxx"}
	 */
	@POST
	@Consumes(MediaType.APPLICATION_JSON)
	public Response registerApplication(@Context final HttpServletRequest httpServletRequest, String jsonString) {
		try {

			logger.info("Bind application. Received JSON string:\n" + jsonString);
			NewAppRequestEntity newAppData = mapper.readValue(jsonString, NewAppRequestEntity.class);

			String appId = newAppData.getAppId();
			String serviceId = newAppData.getServiceId();

			ApplicationManager appManager = ApplicationManagerImpl.getInstance();
			appManager.addApplication(newAppData);

			logger.info("Bound app " + appId + " to service " + serviceId + " for metrics collector.");

			MonitorController controller = MonitorController.getInstance();
			controller.bindService(serviceId, appId);
			controller.addPoller(appId);

			return RestApiResponseHandler.getResponse(Status.CREATED,
					"add_app(" + newAppData.getAppId() + "," + newAppData.getServiceId() + ")");

		} catch (IOException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_parse_JSON_error, e,
					httpServletRequest.getLocale());
		} catch (PolicyNotFoundException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_policy_not_found_error, e,
					httpServletRequest.getLocale());
		} catch (MetricNotSupportedException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_metric_not_supported_error, e,
					httpServletRequest.getLocale());
		} catch (DataStoreException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_database_error, e,
					httpServletRequest.getLocale());
		} catch (MonitorServiceException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_cloud_error, e,
					httpServletRequest.getLocale());
		} catch (Exception e) {
			return RestApiResponseHandler.getResponseError(Response.Status.INTERNAL_SERVER_ERROR, e);
		}
	}

	/**
	 * Remove application
	 */

	@DELETE
	@Consumes(MediaType.APPLICATION_JSON)
	public Response unregisterApplication(@Context final HttpServletRequest httpServletRequest,
			@QueryParam("bindingId") String bindingId, @QueryParam("serviceId") String serviceId,
			@QueryParam("appId") String appId) {
		try {
			logger.info(
					"Remove application. BindingId: " + bindingId + " , appId: " + appId + ", serviceId: " + serviceId);

			ApplicationManager appManager = ApplicationManagerImpl.getInstance();
			appManager.removeApplicationByBindingId(bindingId);

			logger.info("Unbinding app " + appId + " from " + serviceId + " for metrics collector");

			MonitorController.getInstance().unbindService(serviceId, appId);
			MonitorController.getInstance().removePoller(appId);

			return RestApiResponseHandler.getResponse(Status.NO_CONTENT);
		} catch (PolicyNotFoundException e) {

			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_policy_not_found_error, e,
					httpServletRequest.getLocale());
		} catch (DataStoreException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_database_error, e,
					httpServletRequest.getLocale());
		} catch (MonitorServiceException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_cloud_error, e,
					httpServletRequest.getLocale());
		} catch (NoAttachedPolicyException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_no_attached_policy_error, e,
					httpServletRequest.getLocale());
		} catch (Exception e) {
			return RestApiResponseHandler.getResponseError(Response.Status.INTERNAL_SERVER_ERROR, e);
		}
	}

}
