package org.cloudfoundry.autoscaler.rest;

import static org.junit.Assert.assertEquals;

import org.junit.Test;

import com.sun.jersey.api.client.WebResource;
import com.sun.jersey.test.framework.JerseyTest;

public class PolicyRestApiTest extends JerseyTest{
	
	public PolicyRestApiTest()throws Exception {
        super("org.cloudfoundry.autoscaler.rest");
    }
	

	@Test
    public void testHelloWorld() {
        WebResource webResource = resource();
        String responseMsg = webResource.path("/policies/f1a083a1-6fcb-46b9-8bdd-cc94e0a491f1").get(String.class);
        System.out.println("=====" + responseMsg);;
        assertEquals("Hello World", responseMsg);
    }

}
