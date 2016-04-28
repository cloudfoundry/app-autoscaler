package org.cloudfoundry.autoscaler.rest;

import java.io.IOException;
import java.util.List;

import javax.servlet.http.HttpServletRequest;
import javax.ws.rs.GET;
import javax.ws.rs.Path;
import javax.ws.rs.Produces;
import javax.ws.rs.QueryParam;
import javax.ws.rs.core.Context;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.common.util.RestApiResponseHandler;
import org.cloudfoundry.autoscaler.common.util.ValidateUtil;
import org.cloudfoundry.autoscaler.constant.Constants.MESSAGE_KEY;
import org.cloudfoundry.autoscaler.data.couchdb.document.ScalingHistory;
import org.cloudfoundry.autoscaler.exceptions.DataStoreException;
import org.cloudfoundry.autoscaler.manager.ScalingHistoryFilter;
import org.cloudfoundry.autoscaler.manager.ScalingHistoryManager;
import org.json.JSONObject;

import com.fasterxml.jackson.databind.ObjectMapper;

@Path("/history")
public class ScalingHistoryRestApi {
	private static final String CLASS_NAME = ScalingHistoryRestApi.class.getName();
	private static final Logger logger = Logger.getLogger(CLASS_NAME);
	private static final ObjectMapper mapper = new ObjectMapper();

	/**
	 * get history list
	 * 
	 * @param jsonString
	 * @return
	 */
	@GET
	@Produces(MediaType.APPLICATION_JSON)
	public Response getHistoryList(@Context final HttpServletRequest httpServletRequest,
			@QueryParam("appId") String appId, @QueryParam("startTime") String startTime,
			@QueryParam("endTime") String endTime, @QueryParam("status") String status,
			@QueryParam("metrics") String metrics, @QueryParam("offset") String offset,
			@QueryParam("maxCount") String maxCount, @QueryParam("scaleType") String scaleType,
			@QueryParam("timeZone") String timeZone) {
		logger.debug("Get scaling history list. App ID is " + appId);
		try {
			ScalingHistoryFilter filter = new ScalingHistoryFilter();
			filter.setAppId(appId);
			if (!ValidateUtil.isNull(startTime)) {

				filter.setStartTime(Long.parseLong(startTime));
			}
			if (!ValidateUtil.isNull(endTime)) {

				filter.setEndTime(Long.parseLong(endTime));
			}
			if (!ValidateUtil.isNull(scaleType)) {

				filter.setScaleType(scaleType);
			}
			filter.setStatus(status);
			filter.setMetrics(metrics);
			if (maxCount != null)
				filter.setMaxCount(Integer.parseInt(maxCount));
			if (offset != null)
				filter.setOffset(Integer.parseInt(offset));

			ScalingHistoryManager manager = ScalingHistoryManager.getInstance();
			List<ScalingHistory> historyList = manager.getHistoryList(filter, timeZone);
			return RestApiResponseHandler.getResponseOk(mapper.writeValueAsString(historyList));
		} catch (IOException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_build_JSON_error, e,
					httpServletRequest.getLocale());
		} catch (DataStoreException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_database_error, e,
					httpServletRequest.getLocale());
		}
	}

	/**
	 * get history count
	 * 
	 * @param jsonString
	 * @return
	 */
	@Path("/count")
	@GET
	@Produces(MediaType.APPLICATION_JSON)
	public Response getHistoryCount(@Context final HttpServletRequest httpServletRequest,
			@QueryParam("appId") String appId, @QueryParam("startTime") String startTime,
			@QueryParam("endTime") String endTime, @QueryParam("status") String status,
			@QueryParam("metrics") String metrics, @QueryParam("scaleType") String scaleType) {
		logger.debug("Get scaling history count. App ID is " + appId);
		try {
			ScalingHistoryFilter filter = new ScalingHistoryFilter();
			filter.setAppId(appId);
			if (!ValidateUtil.isNull(startTime)) {
				filter.setStartTime(Long.parseLong(startTime));
			}
			if (!ValidateUtil.isNull(endTime)) {
				filter.setEndTime(Long.parseLong(endTime));
			}
			if (!ValidateUtil.isNull(scaleType)) {
				filter.setScaleType(scaleType);
			}
			if (!ValidateUtil.isNull(status)) {
				filter.setStatus(status);
			}
			if (!ValidateUtil.isNull(scaleType)) {
				filter.setScaleType(scaleType);
			}
			ScalingHistoryManager manager = ScalingHistoryManager.getInstance();
			int count = manager.getHistoryCount(filter);
			JSONObject json = new JSONObject();
			json.put("count", count);
			return RestApiResponseHandler.getResponseOk(json);
		} catch (DataStoreException e) {
			return ResponseHelper.getResponseError(MESSAGE_KEY.RestResponseErrorMsg_database_error, e,
					httpServletRequest.getLocale());
		}
	}

}
