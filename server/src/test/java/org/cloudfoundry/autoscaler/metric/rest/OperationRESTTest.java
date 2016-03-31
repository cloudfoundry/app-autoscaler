package org.cloudfoundry.autoscaler.metric.rest;

import static org.cloudfoundry.autoscaler.test.constant.Constants.*;
import static org.junit.Assert.*;

import javax.ws.rs.core.MultivaluedMap;

import org.junit.Test;

import com.sun.jersey.api.client.ClientResponse;
import com.sun.jersey.api.client.WebResource;
import com.sun.jersey.core.util.MultivaluedMapImpl;
import com.sun.jersey.test.framework.JerseyTest;

public class OperationRESTTest extends JerseyTest{
	
	public OperationRESTTest(){
		super("org.cloudfoundry.autoscaler.metric.rest");
	}
	@Test
	public void testSetLogLevel(){
		WebResource webResource = resource();
		MultivaluedMap<String, String> map = new MultivaluedMapImpl();
		map.add("package", "org.cloudfoundry.autoscaler.metric.rest");
		ClientResponse response = webResource.path("/operation/log/INFO").queryParams(map).put(ClientResponse.class);
		assertEquals(response.getStatus(), STATUS200);
	}
	@Test
	public void testGetInfo(){
		WebResource webResource = resource();
		ClientResponse response = webResource.path("/operation/info").get(ClientResponse.class);
		assertEquals(response.getStatus(), STATUS200);
	}
	@Test
	public void testGetConnections(){
		WebResource webResource = resource();
		ClientResponse response = webResource.path("/operation/connection").get(ClientResponse.class);
		assertEquals(response.getStatus(), STATUS200);
	}

}
