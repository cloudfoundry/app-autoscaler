package org.cloudfoundry.autoscaler.common.util;

import javax.ws.rs.core.CacheControl;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

import org.apache.log4j.Logger;
import org.json.JSONObject;

public class RestApiResponseHandler {
	public static final String CLASS_NAME = RestApiResponseHandler.class.getName();
	public static final Logger logger = Logger.getLogger(CLASS_NAME);
	private static CacheControl cc;
	static {
		cc = new CacheControl();
		cc.setNoCache(true);
		cc.setNoTransform(true);
		cc.setPrivate(true);

	}

	public static Response getResponseOk() {
		return Response.ok().cacheControl(cc).build();
	}

	public static Response getResponseOk(String msg) {
		logger.debug(msg);
		return Response.ok(msg).cacheControl(cc).build();
	}

	public static Response getResponseOk(JSONObject jsonObj) {
		String jsonStr = jsonObj.toString();
		logger.debug("Successfully returned JSON string: " + jsonStr);
		return Response.ok(jsonStr, MediaType.APPLICATION_JSON).cacheControl(cc).build();
	}

	public static Response getResponse(javax.ws.rs.core.Response.Status status) {
		return Response.status(status).cacheControl(cc).build();
	}

	public static Response getResponse(javax.ws.rs.core.Response.Status status, String msg) {
		logger.debug(msg);
		return Response.status(status).entity(msg).cacheControl(cc).build();
	}

	public static Response getResponse(javax.ws.rs.core.Response.Status status, JSONObject jsonObj) {
		String jsonStr = jsonObj.toString();
		logger.debug("Successfully returned JSON string: " + jsonStr);
		return Response.status(status).type(MediaType.APPLICATION_JSON).entity(jsonStr).cacheControl(cc).build();
	}

	public static Response getResponse(int status) {
		return Response.status(status).cacheControl(cc).build();
	}

	public static Response getResponse(int status, String msg) {
		return Response.status(status).entity(msg).cacheControl(cc).build();
	}

	public static Response getResponse(int status, JSONObject jsonObj) {
		String jsonStr = jsonObj.toString();
		logger.debug("Successfully returned JSON string: " + jsonStr);
		return Response.status(status).type(MediaType.APPLICATION_JSON).entity(jsonStr).cacheControl(cc).build();
	}

	public static Response getResponseError(javax.ws.rs.core.Response.Status status, Exception e) {
		logger.error(e.getMessage(), e);
		return Response.status(status).type(MediaType.APPLICATION_JSON).entity(getErrorJsonString(e.getMessage()))
				.cacheControl(cc).build();
	}

	public static Response getResponseError(String msg, int statusCode, Exception e) {
		logger.error(msg, e);

		return Response.status(statusCode).type(MediaType.APPLICATION_JSON).entity(getErrorJsonString(msg))
				.cacheControl(cc).build();
	}

	public static Response getResponseUnauthorized(String msg) {
		logger.error(msg);
		return Response.status(Response.Status.UNAUTHORIZED).entity(msg).build();
	}

	public static String getErrorJsonString(String errDesc) {
		return "{\"error\" : \"" + errDesc + "\"}";
	}
}
