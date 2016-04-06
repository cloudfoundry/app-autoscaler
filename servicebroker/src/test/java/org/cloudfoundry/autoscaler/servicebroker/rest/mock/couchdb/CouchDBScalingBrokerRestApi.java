package org.cloudfoundry.autoscaler.servicebroker.rest.mock.couchdb;

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

import org.cloudfoundry.autoscaler.servicebroker.util.RestApiResponseUtil;
import org.json.JSONObject;

@Path("/couchdb-scalingbroker")
public class CouchDBScalingBrokerRestApi {
	//_design/Application_ByAppId/_view/by_appId?key=%221b96f579-29f5-470c-84cc-03dd6ddddb65%22&include_docs=true
	@GET
	@Path("_design/{designDocType}/_view/{viewName}")
	@Produces(MediaType.APPLICATION_JSON)
	public Response getDocument(@Context final HttpServletRequest httpServletRequest,
			@PathParam("designDocType") final String designDocType,@PathParam("viewName") final String viewName,@QueryParam("key") final String key, @QueryParam("include_docs") final String include_docs) {
		String result = this.getResponse(designDocType, viewName);
		return RestApiResponseUtil.getResponseOk(result);

	}
	@GET
	@Path("_design/{designDocType}")
	@Produces(MediaType.APPLICATION_JSON)
	public Response getDesignDocument(@Context final HttpServletRequest httpServletRequest,
			@PathParam("designDocType") final String designDocType) {
		String result = this.getResponse(designDocType);
		return RestApiResponseUtil.getResponseOk(result);

	}
	@GET
	@Path("/")
	@Produces(MediaType.APPLICATION_JSON)
	public Response getDBs(@Context final HttpServletRequest httpServletRequest) {
		String result = "{\"db_name\":\"couchdb-scalingbroker\",\"doc_count\":71,\"doc_del_count\":52,\"update_seq\":427,\"purge_seq\":0,\"compact_running\":false,\"disk_size\":1327211,\"data_size\":56286,\"instance_start_time\":\"1458806210525541\",\"disk_format_version\":6,\"committed_update_seq\":427}";
		return RestApiResponseUtil.getResponseOk(result);

	}

	@DELETE
	@Path("/{docId}")
	@Produces(MediaType.APPLICATION_JSON)
	public Response getSpacesApplications(@Context final HttpServletRequest httpServletRequest,
			@PathParam("docId") String docId, @QueryParam("rev") final String rev) {
		String jsonStr = "{\"ok\": true,\"id\": \"2d642054-5675-44b2-9304-af58b1648365\",\"rev\": \"3-24c637da3fa1164b1a5fe05a35456e1c\"}";
		return RestApiResponseUtil.getResponseOk(jsonStr);

	}

	@PUT
	@Path("/{docId}")
	@Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response updateApplicationInstances(@Context final HttpServletRequest httpServletRequest,
			@PathParam("docId") String docId, String jsonString) {
		
		String returnStr = "{\"ok\": true,\"id\": \"2d642054-5675-44b2-9304-af58b1648365\",\"rev\": \"2-24c637da3fa1164b1a5fe05a35456e1c\"}";
		return RestApiResponseUtil.getResponse(201, returnStr);

	}
	@PUT
	@Path("/")
	@Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response createDB(@Context final HttpServletRequest httpServletRequest,
			@PathParam("docId") String docId, String jsonString) {
		String returnStr = "{\"ok\":true}";
		return RestApiResponseUtil.getResponse(201, returnStr);

	}
	@PUT
	@Path("/_design/{designDocName}")
	@Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response createView(@Context final HttpServletRequest httpServletRequest,
			@PathParam("designDocName") String designDocName, String jsonString) {
		String returnStr = "{\"ok\": true,\"id\": \"2d642054-5675-44b2-9304-af58b1648365\",\"rev\": \"2-24c637da3fa1164b1a5fe05a35456e1c\"}";
		return RestApiResponseUtil.getResponse(201, returnStr);

	}
	private String getResponse(String designDocType){
		return "{\"_id\":\"_design/TriggerRecord_byAll\",\"_rev\":\"1-01b42ba750d48b718e244e9390ce640b\",\"views\":{\"byAll\":{\"map\":\"function(doc) { if (doc.type == 'TriggerRecord' ) emit( [doc.appId, doc.serverName], doc._id )}\"}},\"lists\":{},\"shows\":{},\"language\":\"javascript\",\"filters\":{},\"updates\":{}}";
	}

	private String getResponse(String designDocType, String viewName){
		String resultStr = "";
		switch(designDocType){
		case "ApplicationInstance_ByAppId":
			resultStr = "{\"total_rows\":5,\"offset\":1,\"rows\":[]}";
			//resultStr =  "{\"total_rows\":5,\"offset\":0,\"rows\":[{\"id\":\"3ba3f53e-7181-4dcc-bc51-22cdcb640b32\",\"key\":\"5f3350c6-845a-4c86-9387-f3b2da04e540\",\"value\":\"3ba3f53e-7181-4dcc-bc51-22cdcb640b32\",\"doc\":{\"_id\":\"3ba3f53e-7181-4dcc-bc51-22cdcb640b32\",\"_rev\":\"1-28fe2ae7cd08c99f55f621e75743370a\",\"type\":\"ApplicationInstance_inBroker\",\"bindingId\":\"745a8df7-7635-489a-b9e3-3fb321068acf\",\"serviceId\":\"124f4007-2852-4e65-b221-3cd29a116046\",\"appId\":\"5f3350c6-845a-4c86-9387-f3b2da04e540\",\"timestamp\":1456759218805}}]}";
			break;
		case "ApplicationInstance_ByBindingId":
			resultStr =  "{\"total_rows\":5,\"offset\":0,\"rows\":[{\"id\":\"3ba3f53e-7181-4dcc-bc51-22cdcb640b32\",\"key\":\"5f3350c6-845a-4c86-9387-f3b2da04e540\",\"value\":\"3ba3f53e-7181-4dcc-bc51-22cdcb640b32\",\"doc\":{\"_id\":\"3ba3f53e-7181-4dcc-bc51-22cdcb640b32\",\"_rev\":\"1-28fe2ae7cd08c99f55f621e75743370a\",\"type\":\"ApplicationInstance_inBroker\",\"bindingId\":\"745a8df7-7635-489a-b9e3-3fb321068acf\",\"serviceId\":\"124f4007-2852-4e65-b221-3cd29a116046\",\"appId\":\"5f3350c6-845a-4c86-9387-f3b2da04e540\",\"timestamp\":1456759218805}}]}";
			break;
		case "ServiceInstance_ByServerURL":
			resultStr =  "{\"total_rows\":5,\"offset\":0,\"rows\":[{\"id\":\"2642f56b-8cc4-48c4-a28c-9d3fb2483792\",\"key\":\"http://localhost:9998\",\"value\":\"2642f56b-8cc4-48c4-a28c-9d3fb2483792\"}]}";
			break;
		case "ServiceInstance_ByServiceId":
			resultStr =  "{\"total_rows\":5,\"offset\":0,\"rows\":[{\"id\":\"2642f56b-8cc4-48c4-a28c-9d3fb2483792\",\"key\":\"http://localhost:9998\",\"value\":\"2642f56b-8cc4-48c4-a28c-9d3fb2483792\",\"doc\":{\"_id\":\"2642f56b-8cc4-48c4-a28c-9d3fb2483792\",\"_rev\":\"1-57abaeb827756fc5635abacb88bd4c75\",\"type\":\"ServiceInstance_inBroker\",\"serviceId\":\"f7f74b28-eb49-4acd-9062-d8a0ffa9567e\",\"serverUrl\":\"http://localhost:9998\",\"orgId\":\"4763bd78-ef1b-4c83-9198-13c1d265628e\",\"spaceId\":\"1b7aad51-7469-4e3b-ad84-f140f9bd8d67\",\"timestamp\":1456800826562}}]}";
			break;
		
		
		}
		return resultStr;
	}

}
