package org.cloudfoundry.autoscaler.servicebroker.rest.mock;

import javax.ws.rs.PUT;
import javax.ws.rs.Path;
import javax.ws.rs.core.Response;

@Path("/test")
public class TestMode {

    protected static boolean errorExpected = false;

    @PUT
    @Path("success")
    public Response successTest() {
        errorExpected = false;
        return Response.ok().build();
    }

    @PUT
    @Path("failure")
    public Response failureTest() {
        errorExpected = true;
        return Response.ok().build();
    }

}
