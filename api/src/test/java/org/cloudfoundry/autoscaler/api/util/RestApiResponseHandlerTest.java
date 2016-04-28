package org.cloudfoundry.autoscaler.api.util;

import static org.junit.Assert.assertEquals;

import java.util.Locale;

import javax.servlet.http.HttpServletResponse;
import javax.ws.rs.core.CacheControl;
import javax.ws.rs.core.Response;

import org.cloudfoundry.autoscaler.api.util.MessageUtil;
import org.cloudfoundry.autoscaler.api.util.RestApiResponseHandler;
import org.cloudfoundry.autoscaler.common.InputJSONFormatErrorException;
import org.json.JSONObject;
import org.junit.Test;


public class RestApiResponseHandlerTest {
	private static  CacheControl cc;
	static {
	    cc = new CacheControl();
	    cc.setNoCache(true);
	    cc.setNoTransform(true);
	    cc.setPrivate(true);
	}
	String app_id = "bd653b9e-a7fd-4982-9366-7e233be80156";
	String context = MessageUtil.RestResponseErrorMsg_Create_Update_Policy_context;
	int line = 3;
	int column = 5;
	String errorMessage = "format error";
	
	@Test
	public void getResponseOkTest() {
		assertEquals(RestApiResponseHandler.getResponseOk().getStatus(), HttpServletResponse.SC_OK);
		assertEquals(RestApiResponseHandler.getResponseOk("OK messages").getStatus() , (Response.ok("OK messages").cacheControl(cc).build().getStatus()));
		JSONObject json_obj = new JSONObject();
		json_obj.put("status", 200);
		assertEquals(RestApiResponseHandler.getResponseOk(json_obj).getStatus() , Response.ok(json_obj.toString()).cacheControl(cc).build().getStatus());
	}
	
	@Test
	public void getResponseCreatedOkTest(){
		assertEquals(RestApiResponseHandler.getResponseCreatedOk("Create OK").getStatus(), HttpServletResponse.SC_CREATED);
	}
	
	@Test
	public void getResponseErrorTest() {
		assertEquals(RestApiResponseHandler.getResponseError(new Exception("Internal Server error")).getEntity().toString(), "{\"error\" : \""+"Internal Server error"+"\"}");
	}
    
	@Test 
	public void getResponseInputJsonFormatErrorTest() {
		assertEquals(RestApiResponseHandler.getResponseInputJsonFormatError(new InputJSONFormatErrorException(context, errorMessage, line, column), Locale.US).getEntity().toString(), "{\"error\" : \""+"CWSCV6012E: Format error at line 3 column 5 in the input JSON strings for API: Create/Update Policy."+"\"}");
	}
	    

}
