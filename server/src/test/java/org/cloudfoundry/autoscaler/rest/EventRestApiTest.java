package org.cloudfoundry.autoscaler.rest;

import static org.cloudfoundry.autoscaler.test.constant.Constants.*;
import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNotNull;

import javax.ws.rs.core.MediaType;

import org.cloudfoundry.autoscaler.bean.AutoScalerPolicyTrigger;
import org.cloudfoundry.autoscaler.bean.MonitorTriggerEvent;
import org.cloudfoundry.autoscaler.bean.Trigger;
import org.cloudfoundry.autoscaler.data.couchdb.document.AutoScalerPolicy;
import org.json.JSONObject;
import org.junit.Test;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.sun.jersey.api.client.ClientResponse;
import com.sun.jersey.api.client.WebResource;
import com.sun.jersey.test.framework.JerseyTest;

public class EventRestApiTest extends JerseyTest{
	private static final ObjectMapper mapper = new ObjectMapper();
	private static String policyId = null;
	public EventRestApiTest() throws Exception{
		super("org.cloudfoundry.autoscaler.rest","org.cloudfoundry.autoscaler.rest.mock");
	}
	
	@Test
	public void testReceiveEvents() throws JsonProcessingException{
		
		WebResource webResource = resource();
		String policyStr = "";
		policyStr = PolicyRestApiTest.getPolicyContent();
		ClientResponse response = webResource.path("/policies").type(MediaType.APPLICATION_JSON).post(ClientResponse.class,policyStr);
		String jsonStr = response.getEntity(String.class);
		JSONObject jo = new JSONObject(jsonStr);
		policyId = jo.getString("policyId");
		assertNotNull(policyId);
        assertEquals(response.getStatus(), STATUS201);
        
        webResource = resource();
		jo = new JSONObject();
		jo.put("policyId", policyId);
		jo.put("state", AutoScalerPolicy.STATE_ENABLED);
		response = webResource.path("/apps/" + TESTAPPID).type(MediaType.APPLICATION_JSON).put(ClientResponse.class, jo.toString());
        assertEquals(response.getStatus(), STATUS200);
        
		response = webResource.path("/events").type(MediaType.APPLICATION_JSON).post(ClientResponse.class, mapper.writeValueAsString(this.generateEvents()));
        assertEquals(response.getStatus(), STATUS200);
        
        
	}
	public MonitorTriggerEvent[] generateEvents(){
		
		MonitorTriggerEvent[] array = new MonitorTriggerEvent[1];
		Trigger trigger = new Trigger();		
		trigger.setMetric("Memory");
		trigger.setStatType(Trigger.AGGREGATE_TYPE_MAX);
		trigger.setStatWindowSecs(120);
		trigger.setBreachDurationSecs(120);
		trigger.setUnit("percent");
		trigger.setTriggerId(AutoScalerPolicyTrigger.TriggerId_LowerThreshold);
		trigger.setMetricThreshold(30);
		trigger.setThresholdType(Trigger.THRESHOLD_TYPE_LESS_THAN);
		trigger.setCallbackUrl("http://localhost:8080/resources/events");
		MonitorTriggerEvent event = new MonitorTriggerEvent();
		event.setAppId(TESTAPPID);
		event.setMetricType("Memory");
		event.setMetricValue(1073741824l);
		event.setTimeStamp(System.currentTimeMillis());
		event.setTriggerId("lower");// or upper
		event.setTrigger(trigger);
		array[0] = event;
		return array;
	}
}
