package org.cloudfoundry.autoscaler.rest;

import static org.cloudfoundry.autoscaler.test.constant.Constants.*;
import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNotNull;

import java.io.IOException;

import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.MultivaluedMap;

import org.cloudfoundry.autoscaler.data.couchdb.document.AutoScalerPolicy;
import org.json.JSONObject;
import org.junit.FixMethodOrder;
import org.junit.Test;
import org.junit.runners.MethodSorters;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.sun.jersey.api.client.ClientResponse;
import com.sun.jersey.api.client.WebResource;
import com.sun.jersey.core.util.MultivaluedMapImpl;
import com.sun.jersey.test.framework.JerseyTest;
@FixMethodOrder(MethodSorters.NAME_ASCENDING)
public class ApplicationRestApiTest extends JerseyTest{
	private static String policyId = TESTPOLICYID;
	public ApplicationRestApiTest() throws Exception{
		super("org.cloudfoundry.autoscaler.rest","org.cloudfoundry.autoscaler.rest.mock");
	}
	@Test
	public void test001CreatePolicy() throws IOException{
		WebResource webResource = resource();
		String policyStr = "";
		policyStr = PolicyRestApiTest.getPolicyContent();
		ClientResponse response = webResource.path("/policies").type(MediaType.APPLICATION_JSON).post(ClientResponse.class,policyStr);
		String jsonStr = response.getEntity(String.class);
		JSONObject jo = new JSONObject(jsonStr);
		policyId = jo.getString("policyId");
		assertNotNull(policyId);
        assertEquals(response.getStatus(), STATUS201);
	}
	@Test
	public void test002AttachPolicy() throws JsonProcessingException{
		WebResource webResource = resource();
		JSONObject jo = new JSONObject();
		jo.put("policyId", policyId);
		jo.put("state", AutoScalerPolicy.STATE_ENABLED);
		ClientResponse response = webResource.path("/apps/" + TESTAPPID).type(MediaType.APPLICATION_JSON).put(ClientResponse.class, jo.toString());
        assertEquals(response.getStatus(), STATUS200);
	}
	@Test
	public void test003DetachPolicy() throws JsonProcessingException{
		WebResource webResource = resource();
		MultivaluedMap<String, String> map = new MultivaluedMapImpl();
		map.add("policyId", policyId);
		map.add("state", AutoScalerPolicy.STATE_ENABLED);
		ClientResponse response = webResource.path("/apps/" + TESTAPPID).queryParams(map).type(MediaType.APPLICATION_JSON).delete(ClientResponse.class);
        assertEquals(response.getStatus(), STATUS200);
	}
	@Test
	public void test004GetApplication() throws JsonProcessingException{
		WebResource webResource = resource();
		ClientResponse response = webResource.path("/apps/" + TESTAPPID).type(MediaType.APPLICATION_JSON).get(ClientResponse.class);
        assertEquals(response.getStatus(), STATUS200);
	}
	@Test
	public void test005GetApplications() throws JsonProcessingException{
		WebResource webResource = resource();
		MultivaluedMap<String, String> map = new MultivaluedMapImpl();
		map.add("serviceId", TESTSERVICEID);
		ClientResponse response = webResource.path("/apps").queryParams(map).type(MediaType.APPLICATION_JSON).get(ClientResponse.class);
        assertEquals(response.getStatus(), STATUS201);
	}
	

}
