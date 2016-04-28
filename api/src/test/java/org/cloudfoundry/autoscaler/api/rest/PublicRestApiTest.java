package org.cloudfoundry.autoscaler.api.rest;

import static org.cloudfoundry.autoscaler.api.Constants.STATUS200;
import static org.cloudfoundry.autoscaler.api.Constants.TESTAPPID;
import static org.junit.Assert.assertEquals;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;

import javax.ws.rs.core.MediaType;

import org.json.JSONObject;
import org.junit.Test;

import com.sun.jersey.api.client.ClientResponse;
import com.sun.jersey.api.client.WebResource;
import com.sun.jersey.test.framework.JerseyTest;

public class PublicRestApiTest extends JerseyTest {

	public PublicRestApiTest() throws Exception {
		super("org.cloudfoundry.autoscaler.api.rest", "org.cloudfoundry.autoscaler.api.rest.mock.cc");
	}

	@Test
	public void info() throws Exception {
		WebResource webResource = resource();
		ClientResponse response = webResource.path("/info").type(MediaType.APPLICATION_JSON).get(ClientResponse.class);
		assertEquals(STATUS200, response.getStatus());
		String j = response.getEntity(String.class);
		JSONObject json = new JSONObject(j);
		assertEquals(json.getString("token_endpoint"), "https://localhost:9998");
	}

	@Test
	public void createPolicyTest() throws Exception {
		WebResource webResource = resource();
		String policyStr = getPolicyContent();
		ClientResponse response = webResource.path("/v1/apps/" + TESTAPPID + "/policy").type(MediaType.APPLICATION_JSON)
				.put(ClientResponse.class, policyStr);
		assertEquals(STATUS200, response.getStatus());
	}

	@Test
	public void deletePolicyTest() {
		WebResource webResource = resource();
		ClientResponse response = webResource.path("/v1/apps/" + TESTAPPID + "/policy").type(MediaType.APPLICATION_JSON)
				.delete(ClientResponse.class);
		assertEquals(STATUS200, response.getStatus());
	}

	@Test
	public void getPolicyTest() {
		WebResource webResource = resource();
		ClientResponse response = webResource.path("/v1/apps/" + TESTAPPID + "/policy").type(MediaType.APPLICATION_JSON)
				.get(ClientResponse.class);
		assertEquals(STATUS200, response.getStatus());
	}

	@Test
	public void enablePolicyTest() {
		WebResource webResource = resource();
		ClientResponse response = webResource.path("/v1/apps/" + TESTAPPID + "/policy/status")
				.type(MediaType.APPLICATION_JSON).put(ClientResponse.class, "{ \"enable\": true}");
		assertEquals(STATUS200, response.getStatus());
	}

	@Test
	public void getPolicyStatusTest() {
		WebResource webResource = resource();
		ClientResponse response = webResource.path("/v1/apps/" + TESTAPPID + "/policy/status")
				.type(MediaType.APPLICATION_JSON).get(ClientResponse.class);
		assertEquals(STATUS200, response.getStatus());
	}

	@Test
	public void getHistoryTest() {
		WebResource webResource = resource();
		ClientResponse response = webResource.path("/v1/apps/" + TESTAPPID + "/scalinghistory")
				.type(MediaType.APPLICATION_JSON).get(ClientResponse.class);
		assertEquals(STATUS200, response.getStatus());
	}

	@Test
	public void getMetricsTest() {
		WebResource webResource = resource();
		ClientResponse response = webResource.path("/v1/apps/" + TESTAPPID + "/metrics")
				.type(MediaType.APPLICATION_JSON).get(ClientResponse.class);
		assertEquals(STATUS200, response.getStatus());
	}

	public static String getPolicyContent() {
		BufferedReader br = new BufferedReader(
				new InputStreamReader(PublicRestApiTest.class.getResourceAsStream("/policy.json")));
		String tmp = "";
		String policyStr = "";
		try {
			while ((tmp = br.readLine()) != null) {
				policyStr += tmp;
			}
		} catch (IOException e) {
			e.printStackTrace();
		}
		policyStr = policyStr.replaceAll("\\s+", " ");
		return policyStr;
	}

}
