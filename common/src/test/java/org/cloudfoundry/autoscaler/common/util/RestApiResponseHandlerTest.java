package org.cloudfoundry.autoscaler.common.util;

import static org.junit.Assert.*;



import javax.ws.rs.core.CacheControl;
import javax.ws.rs.core.Response;

import org.json.JSONObject;
import org.junit.Test;

public class RestApiResponseHandlerTest {
	
	@Test
	public void getResponseOkTest() {
		CacheControl cc;
		cc = new CacheControl();
		cc.setNoCache(true);
		cc.setNoTransform(true);
		cc.setPrivate(true);
		assertEquals(RestApiResponseHandler.getResponseOk().getStatus(), 200);
		assertEquals(RestApiResponseHandler.getResponseOk("String for Test").getStatus() , (Response.ok("String for test").cacheControl(cc).build().getStatus()));
		JSONObject jo = new JSONObject();
		jo.put("status", 200);
		assertEquals(RestApiResponseHandler.getResponseOk(jo).getStatus() , Response.ok(jo.toString()).cacheControl(cc).build().getStatus());
	}
	@Test
	public void getResponseTest(){
		
	}

}
