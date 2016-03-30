package org.cloudfoundry.autoscaler.api.rest.mock.serverapi;

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

@Path("/resources/policies")
public class PolicyRestApi {
	private static final String CLASS_NAME = PolicyRestApi.class.getName();
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
    @Consumes(MediaType.APPLICATION_JSON)
	@Produces (MediaType.APPLICATION_JSON)
    public Response createPolicy(@Context final HttpServletRequest httpServletRequest, @QueryParam("org") String orgId, @QueryParam("space") String spaceId, String jsonString)
	{
		
			JSONObject response = new JSONObject();
			response.put("policyId", "testPolicyId");
			return RestApiResponseHandler.getResponseCreatedOk("");
		
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

    /**
     * Gets a policy
     * @param policyId
     * @return
     */
	@GET
	@Path("/{id}")
	@Produces (MediaType.APPLICATION_JSON)
   public Response getPolicy(@Context final HttpServletRequest httpServletRequest, @PathParam("id") String policyId)
	{
			String jsonStr = "";
			return RestApiResponseHandler.getResponseOk(jsonStr);
		
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
