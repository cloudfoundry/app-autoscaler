package org.cloudfoundry.autoscaler.rest;

import java.util.Locale;

import javax.ws.rs.core.Response;

import org.cloudfoundry.autoscaler.common.util.RestApiResponseHandler;
import org.cloudfoundry.autoscaler.constant.Constants.MESSAGE_KEY;
import org.cloudfoundry.autoscaler.util.MessageUtil;

public class ResponseHelper {
	static public Response getResponseError(MESSAGE_KEY key, Exception e, Locale locale) {
		String msg = MessageUtil.getMessageString(key.name(), locale);

		return RestApiResponseHandler.getResponseError(msg, key.getErrorCode(), e);
	}
}
