package org.cloudfoundry.autoscaler.servicebroker.rest.mock;

import java.util.Map;

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
import javax.ws.rs.core.Response.Status;

import org.json.JSONObject;

import com.fasterxml.jackson.databind.ObjectMapper;



@Path("/resources/brokercallback")
public class BrokerCallBackRest {

    private static final ObjectMapper mapper = new ObjectMapper();

    /*
     * Sample request JSON: {"appId": "xxxx", "serviceId": "xxxx", "bindingId": "xxxx"}
     */
    @POST
    @Consumes(MediaType.APPLICATION_JSON)
    public Response registerApplication(@Context final HttpServletRequest httpServletRequest, String jsonString) {
        try {
            Map requestBody = mapper.readValue(jsonString, Map.class);

            String appId = (String) requestBody.get("appId");
            String serviceId = (String) requestBody.get("serviceId");
            String bindingId = (String) requestBody.get("bindingId");
            String agentUsername = (String) requestBody.get("agentUsername");
            String agentPassword = (String) requestBody.get("agentPassword");

            if (!TestMode.errorExpected)
                return Response.status(Status.CREATED).entity("add_app(" + appId + "," + serviceId + ")").build();
            else
                throw new Exception("Test exception");

        } catch (Exception e) {
            return Response.status(Status.INTERNAL_SERVER_ERROR).entity(e.getMessage()).build();
        }
    }

    /**
     * Remove application
     */

    @DELETE
    @Consumes(MediaType.APPLICATION_JSON)
    public Response unregisterApplication(@Context final HttpServletRequest httpServletRequest, @QueryParam("bindingId") String bindingId,
                                          @QueryParam("serviceId") String serviceId, @QueryParam("appId") String appId) {
        try {
            if (!TestMode.errorExpected)
                return Response.status(Status.NO_CONTENT).build();
            else
                throw new Exception("Test exception");

        } catch (Exception e) {
            return Response.status(Status.INTERNAL_SERVER_ERROR).entity(e.getMessage()).build();
        }
    }

    @PUT
    @Path("enablement/{id}")
    @Consumes(MediaType.APPLICATION_JSON)
    public Response tuneEnablement(@Context final HttpServletRequest httpServletRequest, String jsonRequestStr, @PathParam("id") String serviceId) {
        try {
            if (!TestMode.errorExpected)
                return Response.status(Status.NO_CONTENT).build();
            else
                throw new Exception("Test exception");

        } catch (Exception e) {
            return Response.status(Status.INTERNAL_SERVER_ERROR).entity(e.getMessage()).build();
        }
    }

    @GET
    @Path("enablement/{id}")
    @Produces(MediaType.APPLICATION_JSON)
    public Response getEnablementInfo(@Context final HttpServletRequest httpServletRequest, String jsonRequestStr, @PathParam("id") String serviceId) {
        try {
            JSONObject returnResponse = new JSONObject();
            returnResponse.put("state", "STARTED");
            returnResponse.put("enabled", "true");

            if (!TestMode.errorExpected)
                return Response.status(Status.OK).build();
            else
                throw new Exception("Test exception");

        } catch (Exception e) {
            return Response.status(Status.INTERNAL_SERVER_ERROR).entity(e.getMessage()).build();
        }
    }

}
