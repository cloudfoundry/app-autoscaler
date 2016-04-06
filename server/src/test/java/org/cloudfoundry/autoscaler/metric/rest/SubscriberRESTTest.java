package org.cloudfoundry.autoscaler.metric.rest;

import static org.cloudfoundry.autoscaler.test.constant.Constants.*;
import static org.junit.Assert.*;

import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.MultivaluedMap;

import org.cloudfoundry.autoscaler.bean.AutoScalerPolicyTrigger;
import org.cloudfoundry.autoscaler.bean.Trigger;
import org.junit.FixMethodOrder;
import org.junit.Test;
import org.junit.runners.MethodSorters;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.sun.jersey.api.client.ClientHandlerException;
import com.sun.jersey.api.client.ClientResponse;
import com.sun.jersey.api.client.UniformInterfaceException;
import com.sun.jersey.api.client.WebResource;
import com.sun.jersey.core.util.MultivaluedMapImpl;
import com.sun.jersey.test.framework.JerseyTest;
@FixMethodOrder(MethodSorters.NAME_ASCENDING)
public class SubscriberRESTTest extends JerseyTest{
	private static final ObjectMapper mapper = new ObjectMapper();
	public SubscriberRESTTest(){
		super("org.cloudfoundry.autoscaler.metric.rest","org.cloudfoundry.autoscaler.rest.mock");
	}
	@Test
	public void test001AddTrigger() throws UniformInterfaceException, ClientHandlerException, JsonProcessingException{
		WebResource webResource = resource();
		Trigger trigger = new Trigger();
		trigger.setAppId(TESTAPPID);
		trigger.setAppName(TESTAPPNAME);
		trigger.setBreachDurationSecs(120);
		trigger.setCallbackUrl("");
		trigger.setMetric("Memory");
		trigger.setMetricThreshold(30);
		trigger.setStatType(Trigger.AGGREGATE_TYPE_AVG);
		trigger.setStatWindowSecs(120);
		trigger.setThresholdType(Trigger.THRESHOLD_TYPE_LESS_THAN);
		trigger.setTriggerId(AutoScalerPolicyTrigger.TriggerId_LowerThreshold);
		trigger.setUnit("persent");
		ClientResponse response = webResource.path("/triggers").type(MediaType.APPLICATION_JSON).post(ClientResponse.class,mapper.writeValueAsString(trigger));
		assertEquals(response.getStatus(), STATUS200);
	}
	@Test
	public void test002RemoveTrigger(){
		WebResource webResource = resource();
		ClientResponse response = webResource.path("/triggers/" + TESTAPPID).delete(ClientResponse.class);
		assertEquals(response.getStatus(), STATUS200);
	}

}
