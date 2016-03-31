package org.cloudfoundry.autoscaler.rest;

import static org.junit.Assert.assertEquals;

import javax.ws.rs.core.MediaType;

import org.junit.Test;

import com.sun.jersey.api.client.ClientResponse;
import com.sun.jersey.api.client.WebResource;
import com.sun.jersey.test.framework.JerseyTest;

import static org.cloudfoundry.autoscaler.test.constant.Constants.*;

public class ServerStatsRestApiTest extends JerseyTest{
	
	public ServerStatsRestApiTest() throws Exception{
		super("org.cloudfoundry.autoscaler.rest");
	}
	
	@Test
	public void testGetServerStats(){
		WebResource webResource = resource();
		ClientResponse response = webResource.path("/stats").type(MediaType.APPLICATION_JSON).get(ClientResponse.class);
        assertEquals(response.getStatus(), STATUS200);
	}

}
