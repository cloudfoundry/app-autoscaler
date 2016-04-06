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

@Path("/couchdb-scaling")
public class CouchDBScalingRestApi {
	//_design/Application_ByAppId/_view/by_appId?key=%221b96f579-29f5-470c-84cc-03dd6ddddb65%22&include_docs=true
	@GET
	@Path("/_design/{designDocType}/_view/{viewName}")
	@Produces(MediaType.APPLICATION_JSON)
	public Response getDocument(@Context final HttpServletRequest httpServletRequest,
			@PathParam("designDocType") final String designDocType,@PathParam("viewName") final String viewName,@QueryParam("key") final String key, @QueryParam("include_docs") final String include_docs) {
		String result = this.getResponse(designDocType, viewName);
		return RestApiResponseUtil.getResponseOk(result);

	}
	@GET
	@Path("/_design/{designDocType}")
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
		String result = "{\"db_name\":\"couchdb-scaling\",\"doc_count\":71,\"doc_del_count\":52,\"update_seq\":427,\"purge_seq\":0,\"compact_running\":false,\"disk_size\":1327211,\"data_size\":56286,\"instance_start_time\":\"1458806210525541\",\"disk_format_version\":6,\"committed_update_seq\":427}";
		return RestApiResponseUtil.getResponseOk(result);

	}
	@POST
	@Path("/posttest")
	@Consumes(MediaType.APPLICATION_FORM_URLENCODED)
	@Produces(MediaType.APPLICATION_JSON)
	public Response getToken(@Context final HttpServletRequest httpServletRequest, String jsonString) {
		JSONObject jo = new JSONObject();
		jo.put("access_token", "eyJhbGciOiJIUzI1NiJ9");
		jo.put("token_type", "bearer");
		jo.put("expires_in", "43199");
		jo.put("scope","cloud_controller.read cloud_controller.write uaa.resource openid doppler.firehose scim.read cloud_controller.admin");
		jo.put("jti", "0cbe67f1-dd11-4ed3-8b4f-b7e804b9770b");
		return RestApiResponseUtil.getResponseOk(jo.toString());

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
	@Produces(MediaType.APPLICATION_JSON)
	public Response createDB(@Context final HttpServletRequest httpServletRequest) {
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
		case "AppAutoScaleState_ByAppId":
			resultStr =  "{\"total_rows\":3,\"offset\":0,\"rows\":[{\"id\":\"b8c3be6da0037e12ef48bdc252047937\",\"key\":\"ac5b82d6-afeb-4f99-ae74-39e8159da3eb\",\"value\":\"b8c3be6da0037e12ef48bdc252047937\",\"doc\":{\"_id\":\"b8c3be6da0037e12ef48bdc252047937\",\"_rev\":\"6-5aaadd5aeab14c80e40f474ac79218c4\",\"type\":\"AppAutoScaleState\",\"appId\":\"ac5b82d6-afeb-4f99-ae74-39e8159da3eb\",\"instanceCountState\":3,\"lastActionInstanceTarget\":1,\"instanceStepCoolDownEndTime\":0,\"lastActionStartTime\":1459220686013,\"lastActionEndTime\":1459220688081,\"historyId\":\"364b5cb7-79c4-4aa4-9766-65ec135dbe12\"}}]}";
			break;
		case "AppAutoScaleState_byAll":
			resultStr =  "{\"total_rows\":3,\"offset\":0,\"rows\":[{\"id\":\"b8c3be6da0037e12ef48bdc252036764\",\"key\":[\"e4936a31-3dd1-45ba-8961-8d5c38fb3d19\",3],\"value\":\"b8c3be6da0037e12ef48bdc252036764\",\"doc\":{\"_id\":\"b8c3be6da0037e12ef48bdc252036764\",\"_rev\":\"6-11677e80a570b4a5d8664422fac431f0\",\"type\":\"AppAutoScaleState\",\"appId\":\"e4936a31-3dd1-45ba-8961-8d5c38fb3d19\",\"instanceCountState\":3,\"lastActionInstanceTarget\":1,\"instanceStepCoolDownEndTime\":0,\"lastActionStartTime\":1458875975389,\"lastActionEndTime\":1458875976357,\"historyId\":\"c47fb892-73a9-4a3a-a432-2d4ad4ac0bb5\"}}]}";
			break;
		case "Application_ByAppId":
			resultStr =  "{\"total_rows\":12,\"offset\":5,\"rows\":[{\"id\":\"b8c3be6da0037e12ef48bdc2520466f3\",\"key\":\"ac5b82d6-afeb-4f99-ae74-39e8159da3eb\",\"value\":\"b8c3be6da0037e12ef48bdc2520466f3\",\"doc\":{\"_id\":\"b8c3be6da0037e12ef48bdc2520466f3\",\"_rev\":\"10-7b85e10b68782b6eac543fbf72c841aa\",\"type\":\"Application\",\"appId\":\"ac5b82d6-afeb-4f99-ae74-39e8159da3eb\",\"serviceId\":\"4d8e251c-e9c5-4714-9c7f-0245067b4d2d\",\"bindingId\":\"63cf5fb3-7b0d-4673-b1b5-521ee1bb1fd8\",\"appType\":\"java\",\"orgId\":\"380f9890-9333-4e1c-9589-1fa056ebce88\",\"spaceId\":\"9cdab056-bd23-46b4-b8b8-de6943a55686\",\"bindTime\":1459219932743,\"state\":\"enabled\"}}]}";
			break;
		case "Application_ByBindingId":
			resultStr =  "{\"total_rows\":11,\"offset\":0,\"rows\":[{\"id\":\"b8c3be6da0037e12ef48bdc252035232\",\"key\":\"110c00d7-dcd4-4430-8ed0-688886f8484a\",\"value\":\"b8c3be6da0037e12ef48bdc252035232\",\"doc\":{\"_id\":\"b8c3be6da0037e12ef48bdc252035232\",\"_rev\":\"11-5e60e384f007c916a081b72e899d9e7c\",\"type\":\"Application\",\"appId\":\"e4936a31-3dd1-45ba-8961-8d5c38fb3d19\",\"serviceId\":\"4be2d606-7b30-4399-812c-82417bf67575\",\"bindingId\":\"110c00d7-dcd4-4430-8ed0-688886f8484a\",\"appType\":\"java\",\"orgId\":\"380f9890-9333-4e1c-9589-1fa056ebce88\",\"spaceId\":\"9cdab056-bd23-46b4-b8b8-de6943a55686\",\"bindTime\":1458875222320,\"state\":\"unbond\"}}]}";
			break;
		case "Application_ByPolicyId":
			resultStr =  "{\"total_rows\":1,\"offset\":0,\"rows\":[{\"id\":\"b8c3be6da0037e12ef48bdc252366951\",\"key\":\"TESTPOLICYID\",\"value\":\"b8c3be6da0037e12ef48bdc252366951\",\"doc\":{\"_id\":\"b8c3be6da0037e12ef48bdc252366951\",\"_rev\":\"35-c903802db0fd44bc97bf795447b069cc\",\"type\":\"Application\",\"appId\":\"f7e60374-4a81-4fe4-8805-bfc687eeea36\",\"serviceId\":\"TESTSERVICEID\",\"policyId\":\"TESTPOLICYID\",\"policyState\":\"enabled\",\"appType\":\"nodejs\",\"orgId\":\"7f064d05-965c-40d4-b411-d54817c37a6a\",\"spaceId\":\"1563f76e-8544-4c6b-bf26-a4770b4d6579\",\"bindTime\":1459835389366,\"state\":\"enabled\"}}]}";
			break;
		case "Application_ByServiceId":
			resultStr =  "{\"total_rows\":12,\"offset\":0,\"rows\":[{\"id\":\"b8c3be6da0037e12ef48bdc2520037fd\",\"key\":\"14489021-9789-4b50-97c7-e3a847670969\",\"value\":\"b8c3be6da0037e12ef48bdc2520037fd\",\"doc\":{\"_id\":\"b8c3be6da0037e12ef48bdc2520037fd\",\"_rev\":\"4-3d611f41842455767ebd88e10ec461cd\",\"type\":\"Application\",\"appId\":\"927c80bd-bf72-4bf0-891a-7a7415614fbe\",\"serviceId\":\"14489021-9789-4b50-97c7-e3a847670969\",\"bindingId\":\"fe30222f-e310-4a89-be6e-3051bb586f5e\",\"orgId\":\"380f9890-9333-4e1c-9589-1fa056ebce88\",\"spaceId\":\"9cdab056-bd23-46b4-b8b8-de6943a55686\",\"bindTime\":1458806911659,\"state\":\"unbond\"}}]}";
			break;
		case "Application_ByServiceId_State":
			resultStr =  "{\"total_rows\":3,\"offset\":0,\"rows\":[{\"id\":\"b8c3be6da0037e12ef48bdc2520466f3\",\"key\":\"4d8e251c-e9c5-4714-9c7f-0245067b4d2d\",\"value\":\"b8c3be6da0037e12ef48bdc2520466f3\",\"doc\":{\"_id\":\"b8c3be6da0037e12ef48bdc2520466f3\",\"_rev\":\"10-7b85e10b68782b6eac543fbf72c841aa\",\"type\":\"Application\",\"appId\":\"ac5b82d6-afeb-4f99-ae74-39e8159da3eb\",\"serviceId\":\"4d8e251c-e9c5-4714-9c7f-0245067b4d2d\",\"bindingId\":\"63cf5fb3-7b0d-4673-b1b5-521ee1bb1fd8\",\"appType\":\"java\",\"orgId\":\"380f9890-9333-4e1c-9589-1fa056ebce88\",\"spaceId\":\"9cdab056-bd23-46b4-b8b8-de6943a55686\",\"bindTime\":1459219932743,\"state\":\"enabled\"}}]}";
			break;
		case "Application_byAll":
			resultStr =  "{\"total_rows\":12,\"offset\":0,\"rows\":[{\"id\":\"b8c3be6da0037e12ef48bdc2521edeb7\",\"key\":[\"TESTAPPID123456\",\"TESTSERVICEID123456\",\"TESTBINDID123456\",null,\"enabled\"],\"value\":\"b8c3be6da0037e12ef48bdc2521edeb7\",\"doc\":{\"_id\":\"b8c3be6da0037e12ef48bdc2521edeb7\",\"_rev\":\"10-39bb2b57aa77491a9dec6f99ad8320db\",\"type\":\"Application\",\"appId\":\"TESTAPPID123456\",\"serviceId\":\"TESTSERVICEID123456\",\"bindingId\":\"TESTBINDID123456\",\"orgId\":\"TESTORGID123456\",\"spaceId\":\"TESTSPACEID123456\",\"bindTime\":1459325990550,\"state\":\"enabled\"}}]}";
			break;
		case "AutoScalerPolicy_ByPolicyId":
			resultStr =  "{\"total_rows\":18,\"offset\":0,\"rows\":[{\"id\":\"051c5f26-9541-49bc-b155-cca419ae6350\",\"key\":\"051c5f26-9541-49bc-b155-cca419ae6350\",\"value\":\"051c5f26-9541-49bc-b155-cca419ae6350\",\"doc\":{\"_id\":\"051c5f26-9541-49bc-b155-cca419ae6350\",\"_rev\":\"1-b4be650314e0a51011ab13624d3b03cf\",\"type\":\"AutoScalerPolicy\",\"policyId\":\"051c5f26-9541-49bc-b155-cca419ae6350\",\"instanceMinCount\":1,\"instanceMaxCount\":5,\"timezone\":\"(GMT +08:00) Asia/Shanghai\",\"policyTriggers\":[{\"metricType\":\"Memory\",\"statType\":\"average\",\"statWindow\":300,\"breachDuration\":600,\"lowerThreshold\":30,\"upperThreshold\":80,\"instanceStepCountDown\":-1,\"instanceStepCountUp\":2,\"stepDownCoolDownSecs\":600,\"stepUpCoolDownSecs\":600,\"startTime\":0,\"endTime\":0,\"startSetNumInstances\":10,\"endSetNumInstances\":10,\"unit\":\"percent\",\"scaleInAdjustment\":null,\"scaleOutAdjustment\":null}],\"scheduledPolicies\":{}}}]}";
			break;
		case "AutoScalerPolicy_byAll":
			resultStr =  "{\"total_rows\":18,\"offset\":0,\"rows\":[{\"id\":\"f25f9368-3f27-41fb-b338-6becd1dc49ad\",\"key\":[\"f25f9368-3f27-41fb-b338-6becd1dc49ad\",1,5],\"value\":\"f25f9368-3f27-41fb-b338-6becd1dc49ad\",\"doc\":{\"_id\":\"f25f9368-3f27-41fb-b338-6becd1dc49ad\",\"_rev\":\"1-8cacecffa02c9499b47d6ad6e6bec542\",\"type\":\"AutoScalerPolicy\",\"policyId\":\"f25f9368-3f27-41fb-b338-6becd1dc49ad\",\"instanceMinCount\":1,\"instanceMaxCount\":5,\"timezone\":\"(GMT+08:00)Asia/Shanghai\",\"policyTriggers\":[{\"metricType\":\"Memory\",\"statType\":\"average\",\"statWindow\":300,\"breachDuration\":600,\"lowerThreshold\":30,\"upperThreshold\":80,\"instanceStepCountDown\":-1,\"instanceStepCountUp\":2,\"stepDownCoolDownSecs\":600,\"stepUpCoolDownSecs\":600,\"startTime\":0,\"endTime\":0,\"startSetNumInstances\":10,\"endSetNumInstances\":10,\"unit\":\"percent\",\"scaleInAdjustment\":null,\"scaleOutAdjustment\":null}],\"scheduledPolicies\":{}}}]}";
			break;
		case "BoundApp_ByAppId":
			resultStr =  "{\"total_rows\":2,\"offset\":1,\"rows\":[{\"id\":\"b8c3be6da0037e12ef48bdc252200675\",\"key\":[\"TESTAPPID123456\"],\"value\":\"b8c3be6da0037e12ef48bdc252200675\",\"doc\":{\"_id\":\"b8c3be6da0037e12ef48bdc252200675\",\"_rev\":\"1-9004adbcb4bf08a2894c92cb26e9954f\",\"type\":\"BoundApp\",\"appId\":\"TESTAPPID123456\",\"serviceId\":\"TESTSERVICEID123456\",\"serverName\":\"AutoScaling\"}}]}";
			break;
		case "BoundApp_ByServerName":
			resultStr =  "{\"total_rows\":2,\"offset\":0,\"rows\":[{\"id\":\"b8c3be6da0037e12ef48bdc252200675\",\"key\":[\"AutoScaling\"],\"value\":\"b8c3be6da0037e12ef48bdc252200675\",\"doc\":{\"_id\":\"b8c3be6da0037e12ef48bdc252200675\",\"_rev\":\"1-9004adbcb4bf08a2894c92cb26e9954f\",\"type\":\"BoundApp\",\"appId\":\"TESTAPPID123456\",\"serviceId\":\"TESTSERVICEID123456\",\"serverName\":\"AutoScaling\"}}]}";
			break;
		case "BoundApp_ByServiceId":
			resultStr =  "{\"total_rows\":2,\"offset\":0,\"rows\":[{\"id\":\"b8c3be6da0037e12ef48bdc2520477ec\",\"key\":[\"4d8e251c-e9c5-4714-9c7f-0245067b4d2d\"],\"value\":\"b8c3be6da0037e12ef48bdc2520477ec\",\"doc\":{\"_id\":\"b8c3be6da0037e12ef48bdc2520477ec\",\"_rev\":\"2-93b7b43b7edae5e40288a3d5264d32c7\",\"type\":\"BoundApp\",\"appId\":\"ac5b82d6-afeb-4f99-ae74-39e8159da3eb\",\"serviceId\":\"4d8e251c-e9c5-4714-9c7f-0245067b4d2d\",\"appType\":\"java\",\"appName\":\"ScalingTestApp\",\"serverName\":\"AutoScaling\"}}]}";
			break;
		case "BoundApp_ByServiceId_AppId":
			resultStr =  "{\"total_rows\":2,\"offset\":0,\"rows\":[{\"id\":\"b8c3be6da0037e12ef48bdc252200675\",\"key\":[\"TESTSERVICEID123456\",\"TESTAPPID123456\"],\"value\":\"b8c3be6da0037e12ef48bdc252200675\",\"doc\":{\"_id\":\"b8c3be6da0037e12ef48bdc252200675\",\"_rev\":\"1-9004adbcb4bf08a2894c92cb26e9954f\",\"type\":\"BoundApp\",\"appId\":\"TESTAPPID123456\",\"serviceId\":\"TESTSERVICEID123456\",\"serverName\":\"AutoScaling\"}}]}";
			break;
		case "BoundApp_byAll":
			resultStr =  "{\"total_rows\":2,\"offset\":0,\"rows\":[{\"id\":\"b8c3be6da0037e12ef48bdc2520477ec\",\"key\":[\"ac5b82d6-afeb-4f99-ae74-39e8159da3eb\",\"AutoScaling\"],\"value\":\"b8c3be6da0037e12ef48bdc2520477ec\",\"doc\":{\"_id\":\"b8c3be6da0037e12ef48bdc2520477ec\",\"_rev\":\"2-93b7b43b7edae5e40288a3d5264d32c7\",\"type\":\"BoundApp\",\"appId\":\"ac5b82d6-afeb-4f99-ae74-39e8159da3eb\",\"serviceId\":\"4d8e251c-e9c5-4714-9c7f-0245067b4d2d\",\"appType\":\"java\",\"appName\":\"ScalingTestApp\",\"serverName\":\"AutoScaling\"}},{\"id\":\"b8c3be6da0037e12ef48bdc252200675\",\"key\":[\"TESTAPPID123456\",\"AutoScaling\"],\"value\":\"b8c3be6da0037e12ef48bdc252200675\",\"doc\":{\"_id\":\"b8c3be6da0037e12ef48bdc252200675\",\"_rev\":\"1-9004adbcb4bf08a2894c92cb26e9954f\",\"type\":\"BoundApp\",\"appId\":\"TESTAPPID123456\",\"serviceId\":\"TESTSERVICEID123456\",\"serverName\":\"AutoScaling\"}}]}";
			break;
		case "MetricDBSegment_ByPostfix":
			resultStr =  "{\"total_rows\":2,\"offset\":0,\"rows\":[{\"id\":\"b8c3be6da0037e12ef48bdc2520027c6\",\"key\":\"continuously\",\"value\":\"b8c3be6da0037e12ef48bdc2520027c6\",\"doc\":{\"_id\":\"b8c3be6da0037e12ef48bdc2520027c6\",\"_rev\":\"1-8f56ad4bf9cc7c63a71aa1340e25af80\",\"type\":\"MetricDBSegment\",\"startTimestamp\":1458806213698,\"endTimestamp\":9223372036854775807,\"metricDBPostfix\":\"continuously\",\"segmentSeq\":0,\"serverName\":\"AutoScaling\"}},{\"id\":\"b8c3be6da0037e12ef48bdc2520439c2\",\"key\":\"continuously\",\"value\":\"b8c3be6da0037e12ef48bdc2520439c2\",\"doc\":{\"_id\":\"b8c3be6da0037e12ef48bdc2520439c2\",\"_rev\":\"1-362c11a7b1a4e76c45302bf2a3efaf71\",\"type\":\"MetricDBSegment\",\"startTimestamp\":1459152482888,\"endTimestamp\":9223372036854775807,\"metricDBPostfix\":\"continuously\",\"segmentSeq\":0,\"serverName\":\"localhost\"}}]}";
			break;
		case "MetricDBSegment_ByServerName":
			resultStr =  "{\"total_rows\":2,\"offset\":0,\"rows\":[{\"id\":\"b8c3be6da0037e12ef48bdc2520027c6\",\"key\":\"AutoScaling\",\"value\":\"b8c3be6da0037e12ef48bdc2520027c6\",\"doc\":{\"_id\":\"b8c3be6da0037e12ef48bdc2520027c6\",\"_rev\":\"1-8f56ad4bf9cc7c63a71aa1340e25af80\",\"type\":\"MetricDBSegment\",\"startTimestamp\":1458806213698,\"endTimestamp\":9223372036854775807,\"metricDBPostfix\":\"continuously\",\"segmentSeq\":0,\"serverName\":\"AutoScaling\"}}]}";
			break;
		case "MetricDBSegment_ByServerName_SegmentSeq":
			resultStr =  "{\"total_rows\":2,\"offset\":0,\"rows\":[{\"id\":\"b8c3be6da0037e12ef48bdc2520027c6\",\"key\":[\"AutoScaling\",0],\"value\":\"b8c3be6da0037e12ef48bdc2520027c6\",\"doc\":{\"_id\":\"b8c3be6da0037e12ef48bdc2520027c6\",\"_rev\":\"1-8f56ad4bf9cc7c63a71aa1340e25af80\",\"type\":\"MetricDBSegment\",\"startTimestamp\":1458806213698,\"endTimestamp\":9223372036854775807,\"metricDBPostfix\":\"continuously\",\"segmentSeq\":0,\"serverName\":\"AutoScaling\"}}]}";
			break;
		case "MetricDBSegment_byAll":
			resultStr =  "{\"total_rows\":2,\"offset\":0,\"rows\":[{\"id\":\"b8c3be6da0037e12ef48bdc2520027c6\",\"key\":[\"continuously\",\"AutoScaling\",0],\"value\":\"b8c3be6da0037e12ef48bdc2520027c6\",\"doc\":{\"_id\":\"b8c3be6da0037e12ef48bdc2520027c6\",\"_rev\":\"1-8f56ad4bf9cc7c63a71aa1340e25af80\",\"type\":\"MetricDBSegment\",\"startTimestamp\":1458806213698,\"endTimestamp\":9223372036854775807,\"metricDBPostfix\":\"continuously\",\"segmentSeq\":0,\"serverName\":\"AutoScaling\"}},{\"id\":\"b8c3be6da0037e12ef48bdc2520439c2\",\"key\":[\"continuously\",\"localhost\",0],\"value\":\"b8c3be6da0037e12ef48bdc2520439c2\",\"doc\":{\"_id\":\"b8c3be6da0037e12ef48bdc2520439c2\",\"_rev\":\"1-362c11a7b1a4e76c45302bf2a3efaf71\",\"type\":\"MetricDBSegment\",\"startTimestamp\":1459152482888,\"endTimestamp\":9223372036854775807,\"metricDBPostfix\":\"continuously\",\"segmentSeq\":0,\"serverName\":\"localhost\"}}]}";
			break;
		case "ScalingHistory_ByScalingTime":
			resultStr =  "{\"total_rows\":9,\"offset\":0,\"rows\":[{\"id\":\"c47fb892-73a9-4a3a-a432-2d4ad4ac0bb5\",\"key\":[\"e4936a31-3dd1-45ba-8961-8d5c38fb3d19\",1458875975389],\"value\":\"c47fb892-73a9-4a3a-a432-2d4ad4ac0bb5\",\"doc\":{\"_id\":\"c47fb892-73a9-4a3a-a432-2d4ad4ac0bb5\",\"_rev\":\"1-ad4f534c893fb259a5e150067e0ae47a\",\"type\":\"ScalingHistory\",\"appId\":\"e4936a31-3dd1-45ba-8961-8d5c38fb3d19\",\"status\":3,\"adjustment\":-1,\"instances\":1,\"startTime\":1458875975389,\"endTime\":1458875976357,\"trigger\":{\"metrics\":null,\"threshold\":0,\"thresholdType\":null,\"breachDuration\":0,\"triggerType\":1}}}]}";
			break;
		case "ScalingHistory_byAll":
			resultStr =  "{\"total_rows\":9,\"offset\":0,\"rows\":[{\"id\":\"c47fb892-73a9-4a3a-a432-2d4ad4ac0bb5\",\"key\":[\"e4936a31-3dd1-45ba-8961-8d5c38fb3d19\",1458875975389],\"value\":\"c47fb892-73a9-4a3a-a432-2d4ad4ac0bb5\",\"doc\":{\"_id\":\"c47fb892-73a9-4a3a-a432-2d4ad4ac0bb5\",\"_rev\":\"1-ad4f534c893fb259a5e150067e0ae47a\",\"type\":\"ScalingHistory\",\"appId\":\"e4936a31-3dd1-45ba-8961-8d5c38fb3d19\",\"status\":3,\"adjustment\":-1,\"instances\":1,\"startTime\":1458875975389,\"endTime\":1458875976357,\"trigger\":{\"metrics\":null,\"threshold\":0,\"thresholdType\":null,\"breachDuration\":0,\"triggerType\":1}}}]}";
			break;
		case "ServiceConfig_byAll":
			resultStr =  "{\"total_rows\":0,\"offset\":0,\"rows\":[]}";
			break;
		case "TriggerRecord_ByAppId":
			resultStr =  "{\"total_rows\":1,\"offset\":0,\"rows\":[{\"id\":\"b30a4188-b141-4696-80ff-976403bafe70_Memory_lower_30.0\",\"key\":[\"b30a4188-b141-4696-80ff-976403bafe70\"],\"value\":\"b30a4188-b141-4696-80ff-976403bafe70_Memory_lower_30.0\",\"doc\":{\"_id\":\"b30a4188-b141-4696-80ff-976403bafe70_Memory_lower_30.0\",\"_rev\":\"11-72e4e42202d5ee07e075710bb4bc5c47\",\"type\":\"TriggerRecord\",\"appName\":\"\",\"appId\":\"b30a4188-b141-4696-80ff-976403bafe70\",\"trigger\":{\"appName\":\"\",\"appId\":\"b30a4188-b141-4696-80ff-976403bafe70\",\"triggerId\":\"lower\",\"metric\":\"Memory\",\"statWindowSecs\":120,\"breachDurationSecs\":120,\"metricThreshold\":30.0,\"thresholdType\":\"less_than\",\"callbackUrl\":\"http://localhost:9080/com.ibm.ws.icap.autoscaling/resources/events\",\"unit\":\"percent\",\"conditionList\":[],\"statType\":\"avg\"},\"serverName\":\"localhost\"}}]}";
			break;
		case "TriggerRecord_ByServerName":
			resultStr =  "{\"total_rows\":1,\"offset\":0,\"rows\":[{\"id\":\"b30a4188-b141-4696-80ff-976403bafe70_Memory_lower_30.0\",\"key\":[\"b30a4188-b141-4696-80ff-976403bafe70\"],\"value\":\"b30a4188-b141-4696-80ff-976403bafe70_Memory_lower_30.0\",\"doc\":{\"_id\":\"b30a4188-b141-4696-80ff-976403bafe70_Memory_lower_30.0\",\"_rev\":\"11-72e4e42202d5ee07e075710bb4bc5c47\",\"type\":\"TriggerRecord\",\"appName\":\"\",\"appId\":\"b30a4188-b141-4696-80ff-976403bafe70\",\"trigger\":{\"appName\":\"\",\"appId\":\"b30a4188-b141-4696-80ff-976403bafe70\",\"triggerId\":\"lower\",\"metric\":\"Memory\",\"statWindowSecs\":120,\"breachDurationSecs\":120,\"metricThreshold\":30.0,\"thresholdType\":\"less_than\",\"callbackUrl\":\"http://localhost:9080/com.ibm.ws.icap.autoscaling/resources/events\",\"unit\":\"percent\",\"conditionList\":[],\"statType\":\"avg\"},\"serverName\":\"localhost\"}}]}";
			break;
		case "TriggerRecord_byAll":
			resultStr =  "{\"total_rows\":1,\"offset\":0,\"rows\":[{\"id\":\"b30a4188-b141-4696-80ff-976403bafe70_Memory_lower_30.0\",\"key\":[\"b30a4188-b141-4696-80ff-976403bafe70\"],\"value\":\"b30a4188-b141-4696-80ff-976403bafe70_Memory_lower_30.0\",\"doc\":{\"_id\":\"b30a4188-b141-4696-80ff-976403bafe70_Memory_lower_30.0\",\"_rev\":\"11-72e4e42202d5ee07e075710bb4bc5c47\",\"type\":\"TriggerRecord\",\"appName\":\"\",\"appId\":\"b30a4188-b141-4696-80ff-976403bafe70\",\"trigger\":{\"appName\":\"\",\"appId\":\"b30a4188-b141-4696-80ff-976403bafe70\",\"triggerId\":\"lower\",\"metric\":\"Memory\",\"statWindowSecs\":120,\"breachDurationSecs\":120,\"metricThreshold\":30.0,\"thresholdType\":\"less_than\",\"callbackUrl\":\"http://localhost:9080/com.ibm.ws.icap.autoscaling/resources/events\",\"unit\":\"percent\",\"conditionList\":[],\"statType\":\"avg\"},\"serverName\":\"localhost\"}}]}";
			break;
		
		}
		return resultStr;
	}

}
