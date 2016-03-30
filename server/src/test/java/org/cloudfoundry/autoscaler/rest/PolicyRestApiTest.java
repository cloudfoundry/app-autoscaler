package org.cloudfoundry.autoscaler.rest;

import static org.junit.Assert.assertEquals;
import static org.mockito.Mockito.mock;

import java.awt.List;

import static org.mockito.Mockito.*;
import org.junit.Test;

import com.sun.jersey.api.client.WebResource;
import com.sun.jersey.api.client.WebResource.Builder;
import com.sun.jersey.test.framework.JerseyTest;

public class PolicyRestApiTest extends JerseyTest{
	private WebResource builder = null;
	public PolicyRestApiTest()throws Exception {
        super("org.cloudfoundry.autoscaler.rest");
        builder = mock(WebResource.class);
    }
	

	@Test
    public void testHelloWorld() {
       List list = mock(List.class);
       list.add("sss");
       verify(list).remove(anyString());
    }

}
