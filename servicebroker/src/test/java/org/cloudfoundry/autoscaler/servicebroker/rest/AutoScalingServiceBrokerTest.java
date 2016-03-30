package org.cloudfoundry.autoscaler.servicebroker.rest;

import static org.junit.Assert.assertEquals;

import javax.ws.rs.core.MediaType;

import org.json.JSONObject;
import org.junit.FixMethodOrder;
import org.junit.Test;
import org.junit.runners.MethodSorters;
import com.sun.jersey.api.client.ClientResponse;
import com.sun.jersey.api.client.WebResource;
import com.sun.jersey.test.framework.JerseyTest;
@FixMethodOrder(MethodSorters.NAME_ASCENDING)
public class AutoScalingServiceBrokerTest extends JerseyTest{
	
	private final int STATUS200 = 200;
	private final int STATUS201 = 201;
	private final String TESTSERVICEID = "TESTSERVICEID123456";
	private final String TESTORGID = "TESTORGID123456";
	private final String TESTSPACEID = "TESTSPACEID123456";
	private final String TESTAPPID = "TESTAPPID123456";
	private final String TESTBINDID = "TESTBINDID123456";
	public  AutoScalingServiceBrokerTest() throws Exception{
		super("org.cloudfoundry.autoscaler.servicebroker.rest","org.cloudfoundry.autoscaler.servicebroker.rest.mock");
	}
	@Test
	public void test01Catalog() throws Exception{
		WebResource webResource = resource();
		ClientResponse response = webResource.path("/v2/catalog").type(MediaType.APPLICATION_JSON).get(ClientResponse.class);
        assertEquals(response.getStatus(), STATUS200);
	}
	@Test
	public void test02ProvisionService(){
		WebResource webResource = resource();
        JSONObject jo = new JSONObject();
        jo.put("organization_guid", TESTORGID);
        jo.put("space_guid", TESTSPACEID);
        ClientResponse response = webResource.path("/v2/service_instances/" + TESTSERVICEID).type(MediaType.APPLICATION_JSON).put(ClientResponse.class, jo.toString());        
        assertEquals(response.getStatus(), STATUS201);
	}
	@Test
	public void test03BindService(){
		WebResource webResource = resource();
        JSONObject jo = new JSONObject();
        jo.put("app_guid", TESTAPPID);
        ClientResponse response = webResource.path("/v2/service_instances/" + TESTSERVICEID + "/service_bindings/" + TESTBINDID).type(MediaType.APPLICATION_JSON).put(ClientResponse.class,jo.toString());
        assertEquals(response.getStatus(), STATUS200);
	}
	@Test
	public void test04UnbindService(){
		WebResource webResource = resource();
        ClientResponse response = webResource.path("/v2/service_instances/" + TESTSERVICEID + "/service_bindings/" + TESTBINDID).delete(ClientResponse.class);
        assertEquals(response.getStatus(), STATUS200);
	}
	@Test
	public void test05DeprovisionServiceTest(){
		WebResource webResource = resource();
        JSONObject jo = new JSONObject();
        jo.put("organization_guid", TESTORGID);
        jo.put("space_guid", TESTSPACEID);
        ClientResponse response = webResource.path("/v2/service_instances/" + TESTSERVICEID).delete(ClientResponse.class);      
        assertEquals(response.getStatus(), STATUS200);
	}
//	@Test
//	public void test06CallBack(){
//		WebResource webResource = resource();
////		String appId = (String) requestBody.get("appId");
////        String serviceId = (String) requestBody.get("serviceId");
////        String bindingId = (String) requestBody.get("bindingId");
////        String agentUsername = (String) requestBody.get("agentUsername");
////        String agentPassword = (String) requestBody.get("agentPassword");
//        JSONObject jo = new JSONObject();
//        jo.put("appId", "");
//        jo.put("serviceId", "");
//        jo.put("bindingId", "");
//        jo.put("agentUsername", "");
//        jo.put("agentPassword", "");
//        ClientResponse response = webResource.path("/resources/brokercallback").type(MediaType.APPLICATION_JSON).post(ClientResponse.class,jo.toString());      
//        assertEquals(response.getStatus(), STATUS201);
//	}
	

}
