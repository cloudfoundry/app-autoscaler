package org.cloudfoundry.autoscaler.api.rest.mock.cc;

import javax.servlet.http.HttpServletRequest;
import javax.ws.rs.Consumes;
import javax.ws.rs.DELETE;
import javax.ws.rs.GET;
import javax.ws.rs.POST;
import javax.ws.rs.PUT;
import javax.ws.rs.Path;
import javax.ws.rs.PathParam;
import javax.ws.rs.Produces;
import javax.ws.rs.QueryParam;
import javax.ws.rs.core.Context;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.api.util.RestApiResponseHandler;
import org.json.JSONObject;

import com.fasterxml.jackson.databind.ObjectMapper;
@Path("/")
public class CloudControllerRestApi {
	private static final String CLASS_NAME = CloudControllerRestApi.class.getName();
	private static final Logger logger     = Logger.getLogger(CLASS_NAME);
	private ObjectMapper objectMapper = new ObjectMapper();
	/**
	 * Creates a new policy
	 * @param orgId
	 * @param spaceId
	 * @param jsonString
	 * @return
	 */
	@POST
	@Path("/oauth/token")
    @Consumes(MediaType.APPLICATION_JSON)
	@Produces (MediaType.APPLICATION_JSON)
    public Response getToken(@Context final HttpServletRequest httpServletRequest, String jsonString)
	{
//		{
//			"access_token": "eyJhbGciOiJIUzI1NiJ9.eyJqdGkiOiIwY2JlNjKSJjsmJSFAjfsnkJWOAlsgaNSmMS1kZDExLTRlZDMtOGI0Zi1iN2U4MDRiOTc3MGIiLCJzdWIiOiJvZXJ1bnRpbWVfYWRtaW4iLCJhdXRob3JpdGllcyI6WyJjbG91ZF9jb250cm9sbGVyLnJlYWQiLCJjbG91ZF9jb250cm9sbGVyLndyaXRlIiwidWFhLnJlc291cmNlIiwib3BlbmlkIiwiZG9wcGxlci5maXJlaG9zZSIsInNjaW0ucmVhZCIsImNsb3VkX2NvbnRyb2xsZXIuYWRtaW4iXSwic2NvcGUiOlsiY2xvdWRfY29udHJvbGxlci5yZWFkIiwiY2xvdWRfY29udHJvbGxlci53cml0ZSIsInVhYS5yZXNvdXJjZSIsIm9wZW5pZCIsImRvcHBsZXIuZmlyZWhvc2UiLCJzY2ltLnJlYWQiLCJjbG91ZF9jb250cm9sbGVyLmFkbWluIl0sImNsaWVudF9pZCI6Im9lcnVudGltZV9hZG1pbiIsImNpZCI6Im9lcnVudGltZV9hZG1pbiIsImF6cCI6Im9lcnVudGltZV9hZG1pbiIsImdyYW50X3R5cGUiOiJjbGllbnRfY3JlZGVudGlhbHMiLCJyZXZfc2lnIjoiNDkyYjgwOGMiLCJpYXQiOjE0NTkyNDQwNjksImV4cCI6MTQ1OTI4NzI2OSwiaXNzIjoiaHR0cHM6Ly91YWEuc3RhZ2UxLm5nLmJsdWVtaXgubmV0L29hdXRoL3Rva2VuIiwiemlkIjoidWFhIiwiYXVkIjpbIm9lcnVudGltZV9hZG1pbiIsImNsb3VkX2NvbnRyb2xsZXIiLCJ1YWEiLCJvcGVuaWQiLCJkb3BwbGVyIiwic2NpbSJdfQ.cA1kWFYkVu1Ll8I_khJADUONgXh6_ip45yF8PSrxIxc",
//			"token_type": "bearer",
//			"expires_in": 43199,
//			"scope": "cloud_controller.read cloud_controller.write uaa.resource openid doppler.firehose scim.read cloud_controller.admin",
//			"jti": "0cbe67f1-dd11-4ed3-8b4f-b7e804b9770b"
//		}
		
			JSONObject jo = new JSONObject();
			jo.put("access_token", "eyJhbGciOiJIUzI1NiJ9");
			jo.put("token_type", "bearer");
			jo.put("expires_in", "43199");
			jo.put("scope", "cloud_controller.read cloud_controller.write uaa.resource openid doppler.firehose scim.read cloud_controller.admin");
			jo.put("jti", "0cbe67f1-dd11-4ed3-8b4f-b7e804b9770b");
			return RestApiResponseHandler.getResponseCreatedOk(jo.toString());
		
    } 

	/**
	 * Updates a policy
	 * @param policyId
	 * @param jsonString
	 * @return
	 */
	@PUT
	@Path("/{id}")
    @Consumes(MediaType.APPLICATION_JSON)
	@Produces (MediaType.APPLICATION_JSON)
    public Response updatePolicy(@Context final HttpServletRequest httpServletRequest, 
    		@PathParam("id") String policyId, String jsonString)
	{		
			return RestApiResponseHandler.getResponseOk(new JSONObject());
		
    } 

    
	@GET
	@Path("/info")
	@Produces (MediaType.APPLICATION_JSON)
   public Response getCCInfo(@Context final HttpServletRequest httpServletRequest)
	{
//		{
//			"name": "Bluemix",
//			"build": "226002",
//			"support": "http://ibm.com",
//			"version": 2,
//			"description": "IBM Bluemix",
//			"authorization_endpoint": "https://login.stage1.ng.bluemix.net/UAALoginServerWAR",
//			"token_endpoint": "https://uaa.stage1.ng.bluemix.net",
//			"allow_debug": true
//		}
		JSONObject jo = new JSONObject();
		jo.put("name", "openAutoScaler");
		jo.put("build", "1.0");
		jo.put("support", "https://github.com/cfibmers/open-Autoscaler");
		jo.put("version", "1.0");
		jo.put("description", "openAutoScaler");
		jo.put("authorization_endpoint", "http://localhost:9998");
		jo.put("token_endpoint", "https://uaa.stage1.ng.bluemix.net");
		jo.put("allow_debug", "true");
		return RestApiResponseHandler.getResponseOk(jo.toString());
		
	}
		  
    /**
     * Deletes a policy
     * @param org
     * @param space
     * @param id
     * @return
     */
	@DELETE
	@Path("/{id}")
    public Response deletePolicy(@Context final HttpServletRequest httpServletRequest, @PathParam("id")String id)
	{
		
			return RestApiResponseHandler.getResponse204Ok();
    } 
  		

	
    /**
     * Gets brief of policies under the specified service
     * @param serviceId
     * @return
     */
	@GET
	@Produces (MediaType.APPLICATION_JSON)
	public Response getPolicies(@Context final HttpServletRequest httpServletRequest, @QueryParam("service_id") String serviceId) {
				
				return RestApiResponseHandler.getResponseOk();

	}
	
	
}
