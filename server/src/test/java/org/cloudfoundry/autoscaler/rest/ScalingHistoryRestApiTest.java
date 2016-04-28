package org.cloudfoundry.autoscaler.rest;

import static org.cloudfoundry.autoscaler.test.constant.Constants.*;
import static org.junit.Assert.assertEquals;

import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.MultivaluedMap;

import org.cloudfoundry.autoscaler.rest.mock.couchdb.CouchDBDocumentManager;
import org.cloudfoundry.autoscaler.test.testcase.base.JerseyTestBase;
import org.json.JSONArray;
import org.json.JSONObject;
import org.junit.AfterClass;
import org.junit.Test;

import com.sun.jersey.api.client.ClientResponse;
import com.sun.jersey.api.client.WebResource;
import com.sun.jersey.core.util.MultivaluedMapImpl;
import com.sun.jersey.test.framework.JerseyTest;

public class ScalingHistoryRestApiTest extends JerseyTestBase{
	
	public ScalingHistoryRestApiTest() throws Exception{
		super("org.cloudfoundry.autoscaler.rest");
	}
	@Test
	public void testGetHistoryList(){
		WebResource webResource = resource();
		MultivaluedMap<String, String> paramMap = new MultivaluedMapImpl();
		paramMap.add("appId", TESTAPPID);
		paramMap.add("startTime", String.valueOf(0));
		paramMap.add("endTime", String.valueOf(System.currentTimeMillis() * 2));
		paramMap.add("status", "3");
		paramMap.add("scaleType", "scaleIn");
		ClientResponse response = webResource.path("/history").queryParams(paramMap).type(MediaType.APPLICATION_JSON).get(ClientResponse.class);
		String jsonStr = response.getEntity(String.class);
		JSONArray ja = new JSONArray(jsonStr);
		assertEquals(ja.length(), 1);
        assertEquals(response.getStatus(), STATUS200);
	}
	@Test
	public void testGetHistoryCount(){
		WebResource webResource = resource();
		MultivaluedMap<String, String> paramMap = new MultivaluedMapImpl();
		paramMap.add("appId", TESTAPPID);
		paramMap.add("startTime", String.valueOf(0));
		paramMap.add("endTime", String.valueOf(System.currentTimeMillis() * 2));
		paramMap.add("status", "3");
		paramMap.add("scaleType", "scaleIn");
		ClientResponse response = webResource.path("/history/count").queryParams(paramMap).type(MediaType.APPLICATION_JSON).get(ClientResponse.class);
		String jsonStr = response.getEntity(String.class);
		JSONObject jo = new JSONObject(jsonStr);
		assertEquals(jo.get("count"),1);
        assertEquals(response.getStatus(), STATUS200);
	}

}
