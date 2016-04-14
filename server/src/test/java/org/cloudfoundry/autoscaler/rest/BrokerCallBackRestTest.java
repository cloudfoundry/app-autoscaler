package org.cloudfoundry.autoscaler.rest;

import static org.cloudfoundry.autoscaler.test.constant.Constants.*;
import static org.junit.Assert.assertEquals;

import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.MultivaluedMap;

import org.cloudfoundry.autoscaler.rest.mock.couchdb.CouchDBDocumentManager;
import org.json.JSONObject;
import org.junit.FixMethodOrder;
import org.junit.Test;
import org.junit.runners.MethodSorters;

import com.sun.jersey.api.client.ClientResponse;
import com.sun.jersey.api.client.WebResource;
import com.sun.jersey.core.util.MultivaluedMapImpl;
import com.sun.jersey.test.framework.JerseyTest;
@FixMethodOrder(MethodSorters.NAME_ASCENDING)
public class BrokerCallBackRestTest extends JerseyTest{
	
	public BrokerCallBackRestTest() throws Exception{
		super("org.cloudfoundry.autoscaler.rest");
	}
	@Override
	public void tearDown() throws Exception{
		super.tearDown();
		CouchDBDocumentManager.getInstance().initDocuments();
	}
	@Test
	public void test001RegisterApplication(){
		WebResource webResource = resource();
		JSONObject jo = new JSONObject();
		jo.put("appId", TESTAPPID);
		jo.put("serviceId", TESTSERVICEID);
		jo.put("bindingid", TESTBINDINGID);
		ClientResponse response = webResource.path("/brokercallback").type(MediaType.APPLICATION_JSON).post(ClientResponse.class, jo.toString());
        assertEquals(response.getStatus(), STATUS201);
	}
	@Test
	public void test002UnregisterApplication(){
		WebResource webResource = resource();
		MultivaluedMap<String, String> map = new MultivaluedMapImpl();
		map.add("appId", TESTAPPID);
		map.add("serviceId", TESTSERVICEID);
		map.add("bindingid", TESTBINDINGID);
		ClientResponse response = webResource.path("/brokercallback").queryParams(map).type(MediaType.APPLICATION_JSON).delete(ClientResponse.class);
        assertEquals(response.getStatus(), STATUS204);
	}

}
