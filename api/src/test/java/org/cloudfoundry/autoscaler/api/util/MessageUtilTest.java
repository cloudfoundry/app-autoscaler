package org.cloudfoundry.autoscaler.api.util;

import static org.junit.Assert.assertEquals;

import java.util.Locale;

import org.cloudfoundry.autoscaler.api.util.MessageUtil;
import org.junit.Test;


/**
 *
 */
public class MessageUtilTest {

    @Test
    public void messageTest() throws InterruptedException {
    	assertEquals("CWSCV6001E: The API server cannot parse the input JSON strings for API: Create/Update Policy.", MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_input_json_parse_error, Locale.US, "Create/Update Policy"));
    	assertEquals("CWSCV6002E: The API server cannot parse the output JSON strings for API: Get Policy.", MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_output_json_parse_error, Locale.US, "Get Policy"));
    	assertEquals("CWSCV6003E: Input JSON strings format error: type error in the input JSON for API: Create/Update Policy.", MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_input_json_format_error, Locale.US, "type error", "Create/Update Policy"));
    	assertEquals("CWSCV6004E: Output JSON strings format error: type mismatch in the output JSON for API: Get Metric Data.", MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_output_json_format_error, Locale.US, "type mismatch", "Get Metric Data"));
    	assertEquals("CWSCV6005E: Internal server error occurred during retrieve application service information.", MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_internal_server_error, Locale.US, "retrieve application service information"));
    	assertEquals("CWSCV6006E: Calling CloudFoundry APIs failed: Get App Env.", MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_cloud_error, Locale.US, "Get App Env"));
    	assertEquals("CWSCV6007E: The application is not found: d76e90be-15fb-43ac-b9d7-78f81918d77a.", MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_app_not_found_error, Locale.US, "d76e90be-15fb-43ac-b9d7-78f81918d77a"));
    	assertEquals("CWSCV6008E: The following error occurred when retrieving information for application 4367e821-af9e-44ac-ba3d-dc2c15efc527: Could not find specfic matching space for app.", MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_app_info_not_found_error, Locale.US, "4367e821-af9e-44ac-ba3d-dc2c15efc527", "Could not find specfic matching space for app"));
    	assertEquals("CWSCV6009E: Service: CF-AutoScaler for App d76e90be-15fb-43ac-b9d7-78f81918d77a is not found.", MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_service_not_found_error, Locale.US, "CF-AutoScaler", "d76e90be-15fb-43ac-b9d7-78f81918d77a"));
    	assertEquals("CWSCV6010E: Policy for App 4367e821-af9e-44ac-ba3d-dc2c15efc527 is not found.", MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_policy_not_exist_error, Locale.US, "4367e821-af9e-44ac-ba3d-dc2c15efc527"));
    	assertEquals("CWSCV6011E: Internal Authentication failed during Enable Policy.", MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_internal_authentication_failed_error, Locale.US, "Enable Policy"));
    	assertEquals("CWSCV6012E: Format error at line 3 column 5 in the input JSON strings for API: Create/Update Policy.", MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_input_json_format_location_error, Locale.US, "Create/Update Policy", "3", "5"));
    	
    }

  
}