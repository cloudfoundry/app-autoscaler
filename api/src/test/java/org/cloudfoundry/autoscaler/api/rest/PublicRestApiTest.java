package org.cloudfoundry.autoscaler.api.rest;

import org.junit.Test;

import static org.junit.Assert.assertEquals;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.util.HashMap;
import java.util.Map;

import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

import org.cloudfoundry.autoscaler.api.util.CloudFoundryManager;
import static org.cloudfoundry.autoscaler.api.test.constant.Constants.*;
import com.sun.jersey.api.client.ClientResponse;
import com.sun.jersey.api.client.WebResource;
import com.sun.jersey.api.client.WebResource.Builder;
import com.sun.jersey.test.framework.JerseyTest;

public class PublicRestApiTest extends JerseyTest{
	
	
	public  PublicRestApiTest() throws Exception{
		super("org.cloudfoundry.autoscaler.api.rest","org.cloudfoundry.autoscaler.api.mock.cc","org.cloudfoundry.autoscaler.api.mock.serverapi");
	}
	@Test
	public void createPolicyTest() throws Exception{
		WebResource webResource = resource();
		String policyStr = getPolicyContent();
		ClientResponse response = webResource.path("/apps/" + TESTAPPID + "/policy").type(MediaType.APPLICATION_JSON).put(ClientResponse.class,policyStr);
        assertEquals(response.getStatus(), STATUS200);
	}
	@Test
	public void deletePolicyTest(){
		WebResource webResource = resource();
		ClientResponse response = webResource.path("/apps/" + TESTAPPID + "/policy").type(MediaType.APPLICATION_JSON).delete(ClientResponse.class);
		assertEquals(response.getStatus(), STATUS200);
	}
	@Test
	public void getPolicyTest(){
		WebResource webResource = resource();
		ClientResponse response = webResource.path("/apps/" + TESTAPPID + "/policy").type(MediaType.APPLICATION_JSON).get(ClientResponse.class);
		assertEquals(response.getStatus(), STATUS200);
	}
	@Test
	public void enablePolicyTest(){
		WebResource webResource = resource();
		ClientResponse response = webResource.path("/apps/" + TESTAPPID + "/policy/status").type(MediaType.APPLICATION_JSON).put(ClientResponse.class,"{ \"enable\": true}");
		assertEquals(response.getStatus(), STATUS200);
	}
	@Test
	public void getPolicyStatusTest(){
		WebResource webResource = resource();
		ClientResponse response = webResource.path("/apps/" + TESTAPPID + "/policy/status").type(MediaType.APPLICATION_JSON).get(ClientResponse.class);
		assertEquals(response.getStatus(), STATUS200);
	}
	@Test
	public void getHistoryTest(){
		WebResource webResource = resource();
		ClientResponse response = webResource.path("/apps/" + TESTAPPID + "/scalinghistory").type(MediaType.APPLICATION_JSON).get(ClientResponse.class);
		assertEquals(response.getStatus(), STATUS200);
	}
	@Test
	public void getMetricsTest(){
		WebResource webResource = resource();
		ClientResponse response = webResource.path("/apps/" + TESTAPPID + "/metrics").type(MediaType.APPLICATION_JSON).get(ClientResponse.class);
		assertEquals(response.getStatus(), STATUS200);
	}
	public static String getPolicyContent(){
		BufferedReader br = new BufferedReader(new InputStreamReader(PublicRestApiTest.class.getResourceAsStream("/policy.json")));
		String tmp = "";
		String policyStr = "";
		try {
			while((tmp = br.readLine()) != null){
				policyStr += tmp;
			}
		} catch (IOException e) {
			// TODO Auto-generated catch block
			e.printStackTrace();
		}
		policyStr = policyStr.replaceAll("\\s+", " ");
		return policyStr;
	}

}
