package org.cloudfoundry.autoscaler.api.rest;

import org.junit.Test;
import org.mockito.Mockito;

import static org.junit.Assert.assertEquals;
import static org.mockito.Mockito.*;

import java.util.HashMap;
import java.util.Map;

import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

import org.cloudfoundry.autoscaler.api.util.CloudFoundryManager;

import com.sun.jersey.api.client.ClientResponse;
import com.sun.jersey.api.client.WebResource;
import com.sun.jersey.api.client.WebResource.Builder;
import com.sun.jersey.test.framework.JerseyTest;

public class PublicRestApiTest extends JerseyTest{
	
	
	public  PublicRestApiTest() throws Exception{
		super("org.cloudfoundry.autoscaler.api.rest","org.cloudfoundry.autoscaler.api.mock.cc");
	}
//	@Test
//	public void createPolicyTest() throws Exception{
//		WebResource webResource = resource();
//        String responseMsg = webResource.path("/apps/").type(MediaType.APPLICATION_JSON).put(String.class,"");
//        System.out.println("=====" + responseMsg);
//	}
	@Test
	public void deletePolicyTest(){
		WebResource webResource = resource();
		ClientResponse responseMsg = webResource.path("/info").type(MediaType.APPLICATION_JSON).get(ClientResponse.class);
        System.out.println("=====" + responseMsg);
	}
	@Test
	public void getPolicyTest(){
		WebResource webResource = resource();
        String responseMsg = webResource.path("/apps/123456/policy").type(MediaType.APPLICATION_JSON).get(String.class);
        System.out.println("=====" + responseMsg);
	}
	@Test
	public void enablePolicyTest(){}
	@Test
	public void getPolicyStatusTest(){}
	@Test
	public void getHistoryTest(){}
	@Test
	public void getMetricsTest(){}

}
