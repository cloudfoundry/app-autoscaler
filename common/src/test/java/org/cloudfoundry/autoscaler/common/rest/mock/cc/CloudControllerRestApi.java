package org.cloudfoundry.autoscaler.common.rest.mock.cc;

import javax.servlet.http.HttpServletRequest;
import javax.ws.rs.Consumes;
import javax.ws.rs.FormParam;
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

import org.cloudfoundry.autoscaler.common.Constants;
import org.cloudfoundry.autoscaler.common.util.ConfigManager;
import org.cloudfoundry.autoscaler.common.util.RestApiResponseHandler;
import org.json.JSONObject;

@Path("/")
public class CloudControllerRestApi {
	public static final String TESTAPPID = "f7e60374-4a81-4fe4-8805-bfc687eeea36";
	public static final String TESTAPPNAME = "test";
	public static final String TESTSERVICEID = "TESTSERVICEID";

	@GET
	@Path("/info")
	@Produces(MediaType.APPLICATION_JSON)
	public Response getCCInfo(@Context final HttpServletRequest httpServletRequest) {
		// {
		// "name": "OpenAutoScaler",
		// "build": "226002",
		// "support": "http://github.com",
		// "version": 2,
		// "description": "OpenAutoScaler",
		// "authorization_endpoint":
		// "https://login.stage1.ng.bluemix.net/UAALoginServerWAR",
		// "token_endpoint": "https://uaa.stage1.ng.bluemix.net",
		// "allow_debug": true
		// }
		JSONObject jo = new JSONObject();
		jo.put("name", "openAutoScaler");
		jo.put("build", "1.0");
		jo.put("support", "https://github.com/cfibmers/open-Autoscaler");
		jo.put("version", "1.0");
		jo.put("description", "openAutoScaler");
		jo.put("authorization_endpoint", "http://localhost:9998");
		jo.put("token_endpoint", "https://localhost:9998");
		jo.put("allow_debug", "true");
		return RestApiResponseHandler.getResponseOk(jo.toString());

	}

	/**
	 * Creates a new policy
	 * 
	 * @param orgId
	 * @param spaceId
	 * @param jsonString
	 * @return
	 */
	@POST
	@Path("/oauth/token")
	@Consumes(MediaType.APPLICATION_FORM_URLENCODED)
	@Produces(MediaType.APPLICATION_JSON)
	public Response getToken(@Context final HttpServletRequest httpServletRequest,
			@FormParam("grant_type") final String grantType, @FormParam("client_id") final String clientId,
			@FormParam("client_secret") final String clientSecret) {
		// {
		// "access_token":
		// "eyJhbGciOiJIUzI1NiJ9.eyJqdGkiOiIwY2JlNjKSJjsmJSFAjfsnkJWOAlsgaNSmMS1kZDExLTRlZDMtOGI0Zi1iN2U4MDRiOTc3MGIiLCJzdWIiOiJvZXJ1bnRpbWVfYWRtaW4iLCJhdXRob3JpdGllcyI6WyJjbG91ZF9jb250cm9sbGVyLnJlYWQiLCJjbG91ZF9jb250cm9sbGVyLndyaXRlIiwidWFhLnJlc291cmNlIiwib3BlbmlkIiwiZG9wcGxlci5maXJlaG9zZSIsInNjaW0ucmVhZCIsImNsb3VkX2NvbnRyb2xsZXIuYWRtaW4iXSwic2NvcGUiOlsiY2xvdWRfY29udHJvbGxlci5yZWFkIiwiY2xvdWRfY29udHJvbGxlci53cml0ZSIsInVhYS5yZXNvdXJjZSIsIm9wZW5pZCIsImRvcHBsZXIuZmlyZWhvc2UiLCJzY2ltLnJlYWQiLCJjbG91ZF9jb250cm9sbGVyLmFkbWluIl0sImNsaWVudF9pZCI6Im9lcnVudGltZV9hZG1pbiIsImNpZCI6Im9lcnVudGltZV9hZG1pbiIsImF6cCI6Im9lcnVudGltZV9hZG1pbiIsImdyYW50X3R5cGUiOiJjbGllbnRfY3JlZGVudGlhbHMiLCJyZXZfc2lnIjoiNDkyYjgwOGMiLCJpYXQiOjE0NTkyNDQwNjksImV4cCI6MTQ1OTI4NzI2OSwiaXNzIjoiaHR0cHM6Ly91YWEuc3RhZ2UxLm5nLmJsdWVtaXgubmV0L29hdXRoL3Rva2VuIiwiemlkIjoidWFhIiwiYXVkIjpbIm9lcnVudGltZV9hZG1pbiIsImNsb3VkX2NvbnRyb2xsZXIiLCJ1YWEiLCJvcGVuaWQiLCJkb3BwbGVyIiwic2NpbSJdfQ.cA1kWFYkVu1Ll8I_khJADUONgXh6_ip45yF8PSrxIxc",
		// "token_type": "bearer",
		// "expires_in": 43199,
		// "scope": "cloud_controller.read cloud_controller.write uaa.resource
		// openid doppler.firehose scim.read cloud_controller.admin",
		// "jti": "0cbe67f1-dd11-4ed3-8b4f-b7e804b9770b"
		// } httpServletRequest.getP

		if ((grantType == null) || (clientId == null) || (clientSecret == null)
				|| !grantType.equals("client_credentials") || !clientId.equals(ConfigManager.get(Constants.CLIENT_ID))
				|| !clientSecret.equals(ConfigManager.get(Constants.CLIENT_SECRET))) {
			return RestApiResponseHandler.getResponseUnauthorized("An Authentication object with grant type:'"
					+ grantType + "', " + "client_id:'" + clientId + "' was not found in the SecurityContext");
		}

		JSONObject jo = new JSONObject();
		jo.put("access_token", "eyJhbGciOiJIUzI1NiJ9");
		jo.put("token_type", "bearer");
		jo.put("expires_in", 43199);
		jo.put("scope",
				"cloud_controller.read cloud_controller.write uaa.resource openid doppler.firehose scim.read cloud_controller.admin");
		jo.put("jti", "0cbe67f1-dd11-4ed3-8b4f-b7e804b9770b");
		return RestApiResponseHandler.getResponseOk(jo.toString());

	}

	@GET
	@Path("/v2/apps/{appId}/stats")
	@Produces(MediaType.APPLICATION_JSON)
	public Response getApplicationStatus(@Context final HttpServletRequest httpServletRequest,
			@PathParam("appId") final String appId) {
		String jsonStr = "{\"0\":{\"state\":\"RUNNING\",\"stats\":{\"name\":\"test\",\"uris\":[\"test.boshlite.com\"],\"host\":\"23.246.234.65\",\"port\":61318,\"uptime\":1078,\"mem_quota\":1073741824,\"disk_quota\":1073741824,\"fds_quota\":16384,\"usage\":{\"time\":\"2016-03-31 06:23:34 +0000\",\"cpu\":0.0013071996223127328,\"mem\":61423616,\"disk\":61964288}}}}";
		return RestApiResponseHandler.getResponseOk(jsonStr);

	}

	@GET
	@Path("/v2/spaces")
	@Produces(MediaType.APPLICATION_JSON)
	public Response getSpaceByApplication(@Context final HttpServletRequest httpServletRequest,
			@QueryParam("q") String query) {
		String jsonStr = "{ \"total_results\": 1,\"total_pages\": 1,\"prev_url\": null,\"next_url\": null,\"resources\": [{\"metadata\": {\"guid\": \"1563f76e-8544-4c6b-bf26-a4770b4d6579\",\"url\": \"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579\",\"created_at\": \"2015-03-19T08:12:35Z\",\"updated_at\": null},\"entity\": {\"name\": \"TESTSPACE\",\"organization_guid\": \"7f064d05-965c-40d4-b411-d54817c37a6a\",\"space_quota_definition_guid\": null,\"allow_ssh\": true,\"organization_url\": \"/v2/organizations/7f064d05-965c-40d4-b411-d54817c37a6a\",\"developers_url\": \"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579/developers\",\"managers_url\": \"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579/managers\",\"auditors_url\": \"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579/auditors\",\"apps_url\": \"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579/apps\",\"routes_url\": \"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579/routes\",\"domains_url\": \"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579/domains\",\"service_instances_url\": \"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579/service_instances\",\"app_events_url\": \"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579/app_events\",\"events_url\": \"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579/events\",\"security_groups_url\": \"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579/security_groups\"}}]}";
		return RestApiResponseHandler.getResponseOk(jsonStr);

	}

	@GET
	@Path("/v2/organizations")
	@Produces(MediaType.APPLICATION_JSON)
	public Response getOrgnazation(@Context final HttpServletRequest httpServletRequest,
			@QueryParam("q") String query) {
		String jsonStr = "{\"total_results\":1,\"total_pages\":1,\"prev_url\":null,\"next_url\":null,\"resources\":[{\"metadata\":{\"guid\":\"7f064d05-965c-40d4-b411-d54817c37a6a\",\"url\":\"/v2/organizations/7f064d05-965c-40d4-b411-d54817c37a6a\",\"created_at\":\"2015-03-19T08:12:34Z\",\"updated_at\":null},\"entity\":{\"name\":\"TESTORG\",\"billing_enabled\":false,\"quota_definition_guid\":\"2934775d-2c62-496b-9332-30fcb804dad4\",\"status\":\"active\",\"quota_definition_url\":\"/v2/quota_definitions/2934775d-2c62-496b-9332-30fcb804dad4\",\"spaces_url\":\"/v2/organizations/7f064d05-965c-40d4-b411-d54817c37a6a/spaces\",\"domains_url\":\"/v2/organizations/7f064d05-965c-40d4-b411-d54817c37a6a/domains\",\"private_domains_url\":\"/v2/organizations/7f064d05-965c-40d4-b411-d54817c37a6a/private_domains\",\"users_url\":\"/v2/organizations/7f064d05-965c-40d4-b411-d54817c37a6a/users\",\"managers_url\":\"/v2/organizations/7f064d05-965c-40d4-b411-d54817c37a6a/managers\",\"billing_managers_url\":\"/v2/organizations/7f064d05-965c-40d4-b411-d54817c37a6a/billing_managers\",\"auditors_url\":\"/v2/organizations/7f064d05-965c-40d4-b411-d54817c37a6a/auditors\",\"app_events_url\":\"/v2/organizations/7f064d05-965c-40d4-b411-d54817c37a6a/app_events\",\"space_quota_definitions_url\":\"/v2/organizations/7f064d05-965c-40d4-b411-d54817c37a6a/space_quota_definitions\"}}]}";
		return RestApiResponseHandler.getResponseOk(jsonStr);

	}

	@GET
	@Path("/v2/apps/{appId}")
	@Produces(MediaType.APPLICATION_JSON)
	public Response getApplicationById(@Context final HttpServletRequest httpServletRequest,
			@PathParam("appId") final String appId) {
		String jsonStr = "{\"metadata\":{\"guid\":\"" + TESTAPPID + "\",\"url\":\"/v2/apps/" + TESTAPPID
				+ "\",\"created_at\":\"2016-03-21T08:39:36Z\",\"updated_at\":\"2016-03-31T02:21:28Z\"},\"entity\":{\"name\":\""
				+ TESTAPPNAME
				+ "\",\"production\":false,\"space_guid\":\"1563f76e-8544-4c6b-bf26-a4770b4d6579\",\"stack_guid\":\"088e0657-127d-4ab9-b8c4-1401d3aa8c10\",\"buildpack\":null,\"detected_buildpack\":\"SDKforNode.js(TM)(ibm-node.js-4.3.2,buildpack-v3.2-20160315-1257)\",\"environment_json\":{},\"memory\":1024,\"instances\":1,\"disk_quota\":1024,\"state\":\"STARTED\",\"version\":\"9245bcf6-5b90-4426-a24e-05be5fa4fb75\",\"command\":null,\"console\":false,\"debug\":null,\"staging_task_id\":\"ba176b4af3584ed9890ec819cd020a17\",\"package_state\":\"STAGED\",\"health_check_type\":\"port\",\"health_check_timeout\":null,\"staging_failed_reason\":null,\"staging_failed_description\":null,\"diego\":false,\"docker_image\":null,\"package_updated_at\":\"2016-03-21T08:39:50Z\",\"detected_start_command\":\"./vendor/initial_startup.rb\",\"enable_ssh\":true,\"docker_credentials_json\":{\"redacted_message\":\"[PRIVATEDATAHIDDEN]\"},\"ports\":null,\"space_url\":\"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579\",\"stack_url\":\"/v2/stacks/088e0657-127d-4ab9-b8c4-1401d3aa8c10\",\"events_url\":\"/v2/apps/"
				+ TESTAPPID + "/events\",\"service_bindings_url\":\"/v2/apps/" + TESTAPPID
				+ "/service_bindings\",\"routes_url\":\"/v2/apps/" + TESTAPPID + "/routes\"}}";
		return RestApiResponseHandler.getResponseOk(jsonStr);

	}

	@GET
	@Path("/v2/apps/{appId}/env")
	@Produces(MediaType.APPLICATION_JSON)
	public Response getApplicationEnvById(@Context final HttpServletRequest httpServletRequest,
			@PathParam("appId") final String appId) {
		String jsonStr = "{\"staging_env_json\":{\"CF_REGION\":\"open:test:test\"},\"running_env_json\":{\"CF_REGION\":\"open:test:test\"},\"environment_json\":{},\"system_env_json\":{\"VCAP_SERVICES\":{\"CF-AutoScaler\":[{\"name\":\"ttt1\",\"label\":\"CF-AutoScaler\",\"tags\":[\"cf_extensions\",\"cf_created\",\"dev_ops\"],\"plan\":\"free\",\"credentials\":{\"agentUsername\":\"agent\",\"api_url\":\"http://localhost:9998\",\"service_id\":\""
				+ TESTSERVICEID + "\",\"app_id\":\"" + TESTAPPID
				+ "\",\"url\":\"http://localhost:9998\",\"agentPassword\":\"858c1958-29cc-4ae5-bb17-c5308b6d0232\"}}]}},\"application_env_json\":{\"VCAP_APPLICATION\":{\"limits\":{\"mem\":1024,\"disk\":1024,\"fds\":16384},\"application_id\":\""
				+ TESTAPPID
				+ "\",\"application_version\":\"9245bcf6-5b90-4426-a24e-05be5fa4fb75\",\"application_name\":\"test\",\"application_uris\":[\"test.boshlite.com\"],\"version\":\"9245bcf6-5b90-4426-a24e-05be5fa4fb75\",\"name\":\"test\",\"space_name\":\"dev\",\"space_id\":\"1563f76e-8544-4c6b-bf26-a4770b4d6579\",\"uris\":[\"test.boshlite.com\"],\"users\":null}}}";
		return RestApiResponseHandler.getResponseOk(jsonStr);

	}

	@GET
	@Path("/v2/apps/{appId}/summary")
	@Produces(MediaType.APPLICATION_JSON)
	public Response getApplicationSummaryById(@Context final HttpServletRequest httpServletRequest,
			@PathParam("appId") final String appId) {
		String jsonStr = "{\"guid\":\"" + TESTAPPID + "\",\"name\":\"" + TESTAPPNAME
				+ "\",\"routes\":[{\"guid\":\"81730881-bcbc-446d-a23d-e2e87b475115\",\"host\":\"" + TESTAPPNAME
				+ "\",\"domain\":{\"guid\":\"80f7b34d-0c35-47d5-b14a-c0974a9d9b8b\",\"name\":\"stage1.mybluemix.net\"}}],\"running_instances\":1,\"services\":[{\"guid\":\""
				+ TESTSERVICEID
				+ "\",\"name\":\"ttt1\",\"bound_app_count\":1,\"last_operation\":{\"type\":\"create\",\"state\":\"succeeded\",\"description\":\"\",\"updated_at\":null,\"created_at\":\"2016-03-21T08:41:09Z\"},\"dashboard_url\":\"https://Scaling4.stage1.ng.bluemix.net/autoScaling.jsp?org_id=7f064d05-965c-40d4-b411-d54817c37a6a&space_id=1563f76e-8544-4c6b-bf26-a4770b4d6579&service_id="
				+ TESTSERVICEID
				+ "\",\"service_plan\":{\"guid\":\"8a5481ca-d1ce-479e-9098-5881a8fb974b\",\"name\":\"free\",\"service\":{\"guid\":\"ea3d5633-89ac-4a92-9c21-c93952021d4b\",\"label\":\"Auto-Scaling\",\"provider\":null,\"version\":null}}}],\"available_domains\":[{\"guid\":\"80f7b34d-0c35-47d5-b14a-c0974a9d9b8b\",\"name\":\"stage1.mybluemix.net\",\"router_group_guid\":null}],\"name\":\""
				+ TESTAPPNAME
				+ "\",\"production\":false,\"space_guid\":\"1563f76e-8544-4c6b-bf26-a4770b4d6579\",\"stack_guid\":\"088e0657-127d-4ab9-b8c4-1401d3aa8c10\",\"buildpack\":null,\"detected_buildpack\":\"SDK for Node.js(TM) (ibm-node.js-4.3.2, buildpack-v3.2-20160315-1257)\",\"environment_json\":{},\"memory\":1024,\"instances\":1,\"disk_quota\":1024,\"state\":\"STARTED\",\"version\":\"9245bcf6-5b90-4426-a24e-05be5fa4fb75\",\"command\":null,\"console\":false,\"debug\":null,\"staging_task_id\":\"ba176b4af3584ed9890ec819cd020a17\",\"package_state\":\"STAGED\",\"health_check_type\":\"port\",\"health_check_timeout\":null,\"staging_failed_reason\":null,\"staging_failed_description\":null,\"diego\":false,\"docker_image\":null,\"package_updated_at\":\"2016-03-21T08:39:50Z\",\"detected_start_command\":\"./vendor/initial_startup.rb\",\"enable_ssh\":true,\"docker_credentials_json\":{\"redacted_message\":\"[PRIVATE DATA HIDDEN]\"},\"ports\":null}";
		return RestApiResponseHandler.getResponseOk(jsonStr);

	}

	@GET
	@Path("/v2/organizations/{orgnazationId}/spaces")
	@Produces(MediaType.APPLICATION_JSON)
	public Response getOrgnazationSpaces(@Context final HttpServletRequest httpServletRequest,
			@PathParam("orgnazationId") final String orgnazationId) {
		String jsonStr = "{\"total_results\":3,\"total_pages\":1,\"prev_url\":null,\"next_url\":null,\"resources\":[{\"metadata\":{\"guid\":\"1563f76e-8544-4c6b-bf26-a4770b4d6579\",\"url\":\"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579\",\"created_at\":\"2015-03-19T08:12:35Z\",\"updated_at\":null},\"entity\":{\"name\":\"TESTSPACE\",\"organization_guid\":\"7f064d05-965c-40d4-b411-d54817c37a6a\",\"space_quota_definition_guid\":null,\"allow_ssh\":true,\"organization_url\":\"/v2/organizations/7f064d05-965c-40d4-b411-d54817c37a6a\",\"developers_url\":\"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579/developers\",\"managers_url\":\"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579/managers\",\"auditors_url\":\"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579/auditors\",\"apps_url\":\"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579/apps\",\"routes_url\":\"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579/routes\",\"domains_url\":\"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579/domains\",\"service_instances_url\":\"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579/service_instances\",\"app_events_url\":\"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579/app_events\",\"events_url\":\"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579/events\",\"security_groups_url\":\"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579/security_groups\"}},{\"metadata\":{\"guid\":\"0cefc563-2def-426c-bcf0-e3c93822d8d1\",\"url\":\"/v2/spaces/0cefc563-2def-426c-bcf0-e3c93822d8d1\",\"created_at\":\"2015-06-02T09:31:07Z\",\"updated_at\":null},\"entity\":{\"name\":\"autoScalingSp\",\"organization_guid\":\"7f064d05-965c-40d4-b411-d54817c37a6a\",\"space_quota_definition_guid\":null,\"allow_ssh\":true,\"organization_url\":\"/v2/organizations/7f064d05-965c-40d4-b411-d54817c37a6a\",\"developers_url\":\"/v2/spaces/0cefc563-2def-426c-bcf0-e3c93822d8d1/developers\",\"managers_url\":\"/v2/spaces/0cefc563-2def-426c-bcf0-e3c93822d8d1/managers\",\"auditors_url\":\"/v2/spaces/0cefc563-2def-426c-bcf0-e3c93822d8d1/auditors\",\"apps_url\":\"/v2/spaces/0cefc563-2def-426c-bcf0-e3c93822d8d1/apps\",\"routes_url\":\"/v2/spaces/0cefc563-2def-426c-bcf0-e3c93822d8d1/routes\",\"domains_url\":\"/v2/spaces/0cefc563-2def-426c-bcf0-e3c93822d8d1/domains\",\"service_instances_url\":\"/v2/spaces/0cefc563-2def-426c-bcf0-e3c93822d8d1/service_instances\",\"app_events_url\":\"/v2/spaces/0cefc563-2def-426c-bcf0-e3c93822d8d1/app_events\",\"events_url\":\"/v2/spaces/0cefc563-2def-426c-bcf0-e3c93822d8d1/events\",\"security_groups_url\":\"/v2/spaces/0cefc563-2def-426c-bcf0-e3c93822d8d1/security_groups\"}},{\"metadata\":{\"guid\":\"5f47fb12-7543-45d5-844a-6cab43b9bfef\",\"url\":\"/v2/spaces/5f47fb12-7543-45d5-844a-6cab43b9bfef\",\"created_at\":\"2015-08-27T06:47:04Z\",\"updated_at\":null},\"entity\":{\"name\":\"dev2\",\"organization_guid\":\"7f064d05-965c-40d4-b411-d54817c37a6a\",\"space_quota_definition_guid\":null,\"allow_ssh\":true,\"organization_url\":\"/v2/organizations/7f064d05-965c-40d4-b411-d54817c37a6a\",\"developers_url\":\"/v2/spaces/5f47fb12-7543-45d5-844a-6cab43b9bfef/developers\",\"managers_url\":\"/v2/spaces/5f47fb12-7543-45d5-844a-6cab43b9bfef/managers\",\"auditors_url\":\"/v2/spaces/5f47fb12-7543-45d5-844a-6cab43b9bfef/auditors\",\"apps_url\":\"/v2/spaces/5f47fb12-7543-45d5-844a-6cab43b9bfef/apps\",\"routes_url\":\"/v2/spaces/5f47fb12-7543-45d5-844a-6cab43b9bfef/routes\",\"domains_url\":\"/v2/spaces/5f47fb12-7543-45d5-844a-6cab43b9bfef/domains\",\"service_instances_url\":\"/v2/spaces/5f47fb12-7543-45d5-844a-6cab43b9bfef/service_instances\",\"app_events_url\":\"/v2/spaces/5f47fb12-7543-45d5-844a-6cab43b9bfef/app_events\",\"events_url\":\"/v2/spaces/5f47fb12-7543-45d5-844a-6cab43b9bfef/events\",\"security_groups_url\":\"/v2/spaces/5f47fb12-7543-45d5-844a-6cab43b9bfef/security_groups\"}}]}";
		return RestApiResponseHandler.getResponseOk(jsonStr);

	}

	@GET
	@Path("/v2/spaces/{spaceId}/apps")
	@Produces(MediaType.APPLICATION_JSON)
	public Response getSpacesApplications(@Context final HttpServletRequest httpServletRequest,
			@PathParam("spaceId") final String spaceId) {
		String jsonStr = "{\"total_results\":5,\"total_pages\":1,\"prev_url\":null,\"next_url\":null,\"resources\":[{\"metadata\":{\"guid\":\""
				+ TESTAPPID + "\",\"url\":\"/v2/apps/" + TESTAPPID
				+ "\",\"created_at\":\"2016-03-21T08:39:36Z\",\"updated_at\":\"2016-03-31T02:21:28Z\"},\"entity\":{\"name\":\""
				+ TESTAPPNAME
				+ "\",\"production\":false,\"space_guid\":\"1563f76e-8544-4c6b-bf26-a4770b4d6579\",\"stack_guid\":\"088e0657-127d-4ab9-b8c4-1401d3aa8c10\",\"buildpack\":null,\"detected_buildpack\":\"SDKforNode.js(TM)(ibm-node.js-4.3.2,buildpack-v3.2-20160315-1257)\",\"environment_json\":{},\"memory\":1024,\"instances\":1,\"disk_quota\":1024,\"state\":\"STARTED\",\"version\":\"9245bcf6-5b90-4426-a24e-05be5fa4fb75\",\"command\":null,\"console\":false,\"debug\":null,\"staging_task_id\":\"ba176b4af3584ed9890ec819cd020a17\",\"package_state\":\"STAGED\",\"health_check_type\":\"port\",\"health_check_timeout\":null,\"staging_failed_reason\":null,\"staging_failed_description\":null,\"diego\":false,\"docker_image\":null,\"package_updated_at\":\"2016-03-21T08:39:50Z\",\"detected_start_command\":\"./vendor/initial_startup.rb\",\"enable_ssh\":true,\"docker_credentials_json\":{\"redacted_message\":\"[PRIVATEDATAHIDDEN]\"},\"ports\":null,\"space_url\":\"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579\",\"stack_url\":\"/v2/stacks/088e0657-127d-4ab9-b8c4-1401d3aa8c10\",\"events_url\":\"/v2/apps/"
				+ TESTAPPID + "/events\",\"service_bindings_url\":\"/v2/apps/" + TESTAPPID
				+ "/service_bindings\",\"routes_url\":\"/v2/apps/" + TESTAPPID + "/routes\"}}]}";
		return RestApiResponseHandler.getResponseOk(jsonStr);

	}

	@PUT
	@Path("/v2/apps/{appId}")
	@Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response updateApplicationInstances(@Context final HttpServletRequest httpServletRequest,
			@PathParam("appId") String appId, @QueryParam("instances") String instances, String jsonString) {
		String jsonStr = "{\"metadata\":{\"guid\":\"" + TESTAPPID + "\",\"url\":\"/v2/apps/" + TESTAPPID
				+ "\",\"created_at\":\"2016-03-21T08:39:36Z\",\"updated_at\":\"2016-03-31T07:24:33Z\"},\"entity\":{\"name\":\""
				+ TESTAPPNAME
				+ "\",\"production\":false,\"space_guid\":\"1563f76e-8544-4c6b-bf26-a4770b4d6579\",\"stack_guid\":\"088e0657-127d-4ab9-b8c4-1401d3aa8c10\",\"buildpack\":null,\"detected_buildpack\":\"SDKforNode.js(TM)(ibm-node.js-4.3.2,buildpack-v3.2-20160315-1257)\",\"environment_json\":{},\"memory\":1024,\"instances\":1,\"disk_quota\":1024,\"state\":\"STARTED\",\"version\":\"9245bcf6-5b90-4426-a24e-05be5fa4fb75\",\"command\":null,\"console\":false,\"debug\":null,\"staging_task_id\":\"ba176b4af3584ed9890ec819cd020a17\",\"package_state\":\"STAGED\",\"health_check_type\":\"port\",\"health_check_timeout\":null,\"staging_failed_reason\":null,\"staging_failed_description\":null,\"diego\":false,\"docker_image\":null,\"package_updated_at\":\"2016-03-21T08:39:50Z\",\"detected_start_command\":\"./vendor/initial_startup.rb\",\"enable_ssh\":true,\"docker_credentials_json\":{\"redacted_message\":\"[PRIVATEDATAHIDDEN]\"},\"ports\":null,\"space_url\":\"/v2/spaces/1563f76e-8544-4c6b-bf26-a4770b4d6579\",\"stack_url\":\"/v2/stacks/088e0657-127d-4ab9-b8c4-1401d3aa8c10\",\"events_url\":\"/v2/apps/"
				+ TESTAPPID + "/events\",\"service_bindings_url\":\"/v2/apps/" + TESTAPPID
				+ "/service_bindings\",\"routes_url\":\"/v2/apps/" + TESTAPPID + "/routes\"}}";
		return RestApiResponseHandler.getResponseOk(jsonStr);

	}

}
