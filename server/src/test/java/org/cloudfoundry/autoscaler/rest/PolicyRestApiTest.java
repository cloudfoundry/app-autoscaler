package org.cloudfoundry.autoscaler.rest;

import static org.cloudfoundry.autoscaler.test.constant.Constants.*;
import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNotNull;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;

import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.MultivaluedMap;

import org.cloudfoundry.autoscaler.rest.mock.couchdb.CouchDBDocumentManager;
import org.json.JSONArray;
import org.json.JSONObject;
import org.junit.FixMethodOrder;
import org.junit.Test;
import org.junit.runners.MethodSorters;

import com.sun.jersey.api.client.ClientResponse;
import com.sun.jersey.api.client.WebResource;
import com.sun.jersey.core.util.MultivaluedMapImpl;
import com.sun.jersey.test.framework.JerseyTest;
@FixMethodOrder(MethodSorters.NAME_ASCENDING)
public class PolicyRestApiTest extends JerseyTest{
	
	private static String policyId = null;
	public PolicyRestApiTest()throws Exception {
        super("org.cloudfoundry.autoscaler.rest");
    }
	@Override
	public void tearDown() throws Exception{
		super.tearDown();
		CouchDBDocumentManager.getInstance().initDocuments();
	}
	@Test
	public void test001CreatePolicy() throws IOException{
		WebResource webResource = resource();
		String policyStr = "";
		policyStr = getPolicyContent();
		ClientResponse response = webResource.path("/policies").type(MediaType.APPLICATION_JSON).post(ClientResponse.class,policyStr);
		String jsonStr = response.getEntity(String.class);
		JSONObject jo = new JSONObject(jsonStr);
		policyId = jo.getString("policyId");
		assertNotNull(policyId);
        assertEquals(response.getStatus(), STATUS201);
	}
	@Test
	public void test002UpdatePolicy() throws IOException{
		WebResource webResource = resource();
		String policyStr = "";
		policyStr = getPolicyContent();
		ClientResponse response = webResource.path("/policies/" + policyId).type(MediaType.APPLICATION_JSON).put(ClientResponse.class,policyStr);
        assertEquals(response.getStatus(), STATUS200);
	}
	@Test
	public void test003GetPolicy(){
		WebResource webResource = resource();
		ClientResponse response = webResource.path("/policies/" + policyId).type(MediaType.APPLICATION_JSON).get(ClientResponse.class);
        assertEquals(response.getStatus(), STATUS200);
	}
	@Test
	public void test004GetPolicies(){
		WebResource webResource = resource();
		MultivaluedMap<String, String> paramMap = new MultivaluedMapImpl();
		paramMap.add("service_id", TESTSERVICEID);
		ClientResponse response = webResource.path("/policies").queryParams(paramMap).type(MediaType.APPLICATION_JSON).get(ClientResponse.class);
        assertEquals(response.getStatus(), STATUS200);
	}
	@Test
	public void test005DeletePolicy(){
		WebResource webResource = resource();
		ClientResponse response = webResource.path("/policies/" + policyId).type(MediaType.APPLICATION_JSON).delete(ClientResponse.class);
        assertEquals(response.getStatus(), STATUS204);
	}
	public static String getPolicyContent(){
		BufferedReader br = new BufferedReader(new InputStreamReader(PolicyRestApiTest.class.getResourceAsStream("policy.json")));
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
