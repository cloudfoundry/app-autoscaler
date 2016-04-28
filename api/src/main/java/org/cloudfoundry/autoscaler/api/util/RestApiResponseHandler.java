package org.cloudfoundry.autoscaler.api.util;


import java.util.Locale;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.api.util.MessageUtil;
import org.cloudfoundry.autoscaler.common.AppInfoNotFoundException;
import org.cloudfoundry.autoscaler.common.AppNotFoundException;
import org.cloudfoundry.autoscaler.common.CloudException;
import org.cloudfoundry.autoscaler.common.InputJSONFormatErrorException;
import org.cloudfoundry.autoscaler.common.InputJSONParseErrorException;
import org.cloudfoundry.autoscaler.common.InternalAuthenticationException;
import org.cloudfoundry.autoscaler.common.InternalServerErrorException;
import org.cloudfoundry.autoscaler.common.OutputJSONFormatErrorException;
import org.cloudfoundry.autoscaler.common.OutputJSONParseErrorException;
import org.cloudfoundry.autoscaler.common.PolicyNotFoundException;
import org.cloudfoundry.autoscaler.common.ServiceNotFoundException;

import javax.ws.rs.core.CacheControl;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

import org.json.JSONObject;

public class RestApiResponseHandler {
	public static final String CLASS_NAME = RestApiResponseHandler.class.getName();
	public static final Logger logger     = Logger.getLogger(CLASS_NAME); 
	private static  CacheControl cc;
	static {
	    cc = new CacheControl();
	    cc.setNoCache(true);
	    cc.setNoTransform(true);
	    cc.setPrivate(true);
	}
	
	public static Response getResponseOk()
	{
        return Response.ok().cacheControl(cc).build();
	}
	
	public static Response getResponseOk(String msg)
	{
		logger.info(msg);
        return Response.ok(msg).build();
	}

	public static Response getResponseOk(JSONObject jsonObj)
	{
		String jsonStr = jsonObj.toString();
		logger.debug("Successfully returned JSON string: "+jsonStr);
        return Response.ok(jsonStr,MediaType.APPLICATION_JSON).cacheControl(cc).build();
	}
	/*****************************************************************************************************************
	 * 
	 */
	// public convenience methods
	
	public static Response getResponseCreatedOk(String msg)
	{
		logger.info(msg);
        return Response.status(201).entity(msg).build();
	}

	public static Response getResponse200Ok(String msg)
	{
		logger.info(msg);
        return Response.status(200).entity(msg).build();
	}
	
	public static Response getResponse204Ok()
	{
        return Response.status(204).build();
	}
	
	
	public static Response getResponseJsonOk(String jsonString)
	{
		logger.info("Successfully returned JSON string: "+jsonString);
        return Response.ok(jsonString,MediaType.APPLICATION_JSON).build();
	}

	public static Response getResponseJsonCreatedOk(String jsonString)
	{
		logger.info("Successfully returned JSON string: "+jsonString);
        return Response.status(201).entity(jsonString).build();
	}
	
	public static Response getResponseError(Exception e)
	{
		logger.error(e.getMessage(),e);
        return Response.status(Response.Status.INTERNAL_SERVER_ERROR).entity(getErrorJsonString(e.getMessage())).build();
	}
	
	public static Response getResponseInternalServerError(String msg)
	{
		logger.error(msg);
        return Response.status(Response.Status.INTERNAL_SERVER_ERROR).entity(getErrorJsonString(msg)).build();
	}
	
	public static Response getResponseUnauthorized(String msg)
	{
		logger.error(msg);
        return Response.status(Response.Status.UNAUTHORIZED).entity(msg).build();
	}
	
	public static Response getResponseBadRequest(String msg)
	{
		logger.error(msg);
        return Response.status(Response.Status.BAD_REQUEST).entity(msg).build();
	}
	
	public static Response getResponseNotFound(String msg)
	{
		logger.error(msg);
        return Response.status(Response.Status.NOT_FOUND).entity(msg).build();
	}
	
	public static Response getResponseJsonBuildError(Throwable t, Locale locale)
	{
		String msg = MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_build_JSON_error, locale);
		logger.error(msg,t);
        return Response.status(Response.Status.BAD_REQUEST).entity(getErrorJsonString(msg)).build();
	}
	
	public static Response getResponseJsonParseError(Throwable t, Locale locale)
	{
		String msg = MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_parse_JSON_error, locale);
		logger.error(msg,t);
        return Response.status(Response.Status.BAD_REQUEST).entity(getErrorJsonString(msg)).build();
	}

	public static Response getResponseAppNotFound(AppNotFoundException e, Locale locale)
	{
		String msg = MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_app_not_found_error, locale, e.getAppId());
		logger.warn(msg,e);
        return Response.status(Response.Status.NOT_FOUND).entity(getErrorJsonString(msg)).build();
	}
	
	public static Response getResponseAppInfoNotFound(AppInfoNotFoundException e, Locale locale)
	{
		String msg = MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_app_info_not_found_error, locale, e.getAppId(), e.getMessage());
		logger.warn(msg,e);
        return Response.status(Response.Status.INTERNAL_SERVER_ERROR).entity(getErrorJsonString(msg)).build();
	}

	public static Response getResponseCloudError(CloudException e, Locale locale)
	{
		String msg = MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_cloud_error, locale, e.toString());
		logger.error(msg,e);
        return Response.status(Response.Status.INTERNAL_SERVER_ERROR).entity(getErrorJsonString(msg)).build();
	}

	public static Response getResponseServiceNotFound(ServiceNotFoundException e, Locale locale)
	{
		String msg = MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_service_not_found_error, locale, e.getServiceName(), e.getAppId());
		logger.error(msg,e);
        return Response.status(Response.Status.BAD_REQUEST).entity(getErrorJsonString(msg)).build();
	}
	
	public static Response getResponseInternalServerError(InternalServerErrorException e, Locale locale)
	{
		logger.error(e.getMessage());
		String context = MessageUtil.getMessageString(e.getContext(), locale);
		String msg = MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_internal_server_error, locale, context);
        return Response.status(Response.Status.INTERNAL_SERVER_ERROR).entity(getErrorJsonString(msg)).build();
	}
	
	public static Response getResponseInputJsonParseError(InputJSONParseErrorException e, Locale locale)
	{
		logger.error(e.getMessage());
		String context = MessageUtil.getMessageString(e.getContext(), locale);
		String msg = MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_input_json_parse_error, locale, context);
        return Response.status(Response.Status.BAD_REQUEST).entity(getErrorJsonString(msg)).build();
	}
	
	public static Response getResponseInputJsonFormatError(InputJSONFormatErrorException e, Locale locale)
	{
		logger.error(e.getMessage());
		String context = MessageUtil.getMessageString(e.getContext(), locale);
		String msg;
		if (e.getLine() > 0){ // exception with location information
			msg = MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_input_json_format_location_error, locale, context, e.getLine(), e.getColumn());
		}
		else {	
		    msg = MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_input_json_format_error, locale, e.getMessage(), context);
		}
        return Response.status(Response.Status.BAD_REQUEST).entity(getErrorJsonString(msg)).build();
	}
	
	public static Response getResponseOutputJsonParseError(OutputJSONParseErrorException e, Locale locale)
	{
		logger.error(e.getMessage());
		String context = MessageUtil.getMessageString(e.getContext(), locale);
		String msg = MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_output_json_parse_error, locale, context);
        return Response.status(Response.Status.BAD_REQUEST).entity(getErrorJsonString(msg)).build();
	}
	
	public static Response getResponseOutputJsonFormatError(OutputJSONFormatErrorException e, Locale locale)
	{
		logger.error(e.getMessage());
		String context = MessageUtil.getMessageString(e.getContext(), locale);
		String msg = MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_output_json_format_error, locale, e.getMessage(), context);
        return Response.status(Response.Status.INTERNAL_SERVER_ERROR).entity(getErrorJsonString(msg)).build();
	}
	
	public static Response getResponseInternalAuthenticationFail(InternalAuthenticationException e, Locale locale){
		logger.error(e.getMessage());
		String context = MessageUtil.getMessageString(e.getContext(), locale);
		String msg = MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_internal_authentication_failed_error, locale, context);
        return Response.status(Response.Status.INTERNAL_SERVER_ERROR).entity(getErrorJsonString(msg)).build();
	}
	
	public static Response getResponsePolicyNotExistError(PolicyNotFoundException e, Locale locale)
	{
		logger.error(e.getMessage());
		String msg = MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_policy_not_exist_error, locale, e.getAppId());
        return Response.status(Response.Status.NOT_FOUND).entity(getErrorJsonString(msg)).build();
	}
	
	public static String getErrorJsonString(String errDesc)
	{
		return "{\"error\" : \""+errDesc+"\"}";		
	}
	
	public static String getErrorMessage(Exception e, Locale locale)
	{
		String msg="";
		if(e instanceof AppNotFoundException){
			msg = MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_app_not_found_error, locale, ((AppNotFoundException)e).getAppId());
		}
		else if(e instanceof AppInfoNotFoundException){
			msg = MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_app_info_not_found_error, locale, ((AppInfoNotFoundException)e).getAppId(), ((AppInfoNotFoundException)e).getMessage());
		}	
		
		return msg;
	}
}

