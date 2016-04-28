package org.cloudfoundry.autoscaler.api.rest;

import java.io.IOException;
import java.util.HashMap;
import java.util.Map;
import java.util.Date;

import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import javax.ws.rs.Consumes;
import javax.ws.rs.DELETE;
import javax.ws.rs.GET;
import javax.ws.rs.PUT;
import javax.ws.rs.Path;
import javax.ws.rs.PathParam;
import javax.ws.rs.Produces;
import javax.ws.rs.QueryParam;
import javax.ws.rs.core.Context;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

//support Client API
import com.sun.jersey.api.client.Client;
import com.sun.jersey.api.client.ClientResponse;
import com.sun.jersey.api.client.WebResource;

import org.apache.log4j.Logger;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.JsonNode;

import org.json.JSONObject;
import org.cloudfoundry.autoscaler.api.util.MessageUtil;
import org.cloudfoundry.autoscaler.api.util.RestApiResponseHandler;
import org.cloudfoundry.autoscaler.api.validation.ValidateUtil;
import org.cloudfoundry.autoscaler.api.validation.ValidateUtil.DataType;
import org.cloudfoundry.autoscaler.common.AppInfoNotFoundException;
import org.cloudfoundry.autoscaler.common.AppNotFoundException;
import org.cloudfoundry.autoscaler.common.ClientIDLoginFailedException;
import org.cloudfoundry.autoscaler.common.CloudException;
import org.cloudfoundry.autoscaler.common.Constants;
import org.cloudfoundry.autoscaler.common.InputJSONFormatErrorException;
import org.cloudfoundry.autoscaler.common.InputJSONParseErrorException;
import org.cloudfoundry.autoscaler.common.InternalAuthenticationException;
import org.cloudfoundry.autoscaler.common.InternalServerErrorException;
import org.cloudfoundry.autoscaler.common.OutputJSONFormatErrorException;
import org.cloudfoundry.autoscaler.common.OutputJSONParseErrorException;
import org.cloudfoundry.autoscaler.common.PolicyNotFoundException;
import org.cloudfoundry.autoscaler.common.ServiceNotFoundException;
import org.cloudfoundry.autoscaler.common.util.CloudFoundryManager;
import org.cloudfoundry.autoscaler.common.util.ConfigManager;
import org.cloudfoundry.autoscaler.common.util.LocaleUtil;
import org.cloudfoundry.autoscaler.common.util.RestUtil;

@Path("/v1/apps")
public class PublicRestApi {
	private static final String CLASS_NAME = PublicRestApi.class.getName();
	private static final Logger logger = Logger.getLogger(CLASS_NAME);
	private ObjectMapper objectMapper = new ObjectMapper();
	private static volatile PublicRestApi instance;

	public static PublicRestApi getInstance() throws Exception {
		if (instance == null) {
			synchronized (CloudFoundryManager.class) {
				if (instance == null)
					instance = new PublicRestApi();
			}
		}
		return instance;
	}

	/**
	 * Creates a new policy or update an existing policy
	 * 
	 * @param app_id
	 * @param jsonString
	 * @return
	 */
	@PUT
	@Path("/{app_id}/policy")
	@Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response createPolicy(@Context final HttpServletRequest httpServletRequest,
			@PathParam("app_id") String app_id, String jsonString) {
		logger.info("Received JSON String of policy content: " + jsonString);

		Client client = RestUtil.getHTTPSRestClient();
		String policyId;
		String server_url;
		String request_url;
		Map<String, String> service_info;
		try {
			service_info = getserverinfo(app_id);
			server_url = service_info.get("server_url");
			policyId = service_info.get("policyId");

		} catch (CloudException e) {
			return RestApiResponseHandler.getResponseCloudError(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (AppNotFoundException e) {
			return RestApiResponseHandler.getResponseAppNotFound(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (ServiceNotFoundException e) {
			return RestApiResponseHandler.getResponseServiceNotFound(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (InternalAuthenticationException e) {
			return RestApiResponseHandler.getResponseInternalAuthenticationFail(e,
					LocaleUtil.getLocale(httpServletRequest));
		} catch (ClientIDLoginFailedException e) {
			logger.error("login UAA with client ID " + e.getClientID() + " failed");
			return RestApiResponseHandler.getResponseInternalServerError(MessageUtil.getMessageString(
					MessageUtil.RestResponseErrorMsg_internal_server_error, LocaleUtil.getLocale(httpServletRequest)));
		} catch (Exception e) {
			logger.info("error in getserverinfo: " + e.getMessage());
			return RestApiResponseHandler.getResponseInternalServerError(
					new InternalServerErrorException(
							MessageUtil.RestResponseErrorMsg_retrieve_application_service_information_context, e),
					LocaleUtil.getLocale(httpServletRequest));
		}
		// transform the input
		Map<String, String> orgSpace;
		try {
			orgSpace = CloudFoundryManager.getInstance().getOrgSpaceByAppId(app_id);
		} catch (CloudException e) {
			return RestApiResponseHandler.getResponseCloudError(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (AppNotFoundException e) {
			return RestApiResponseHandler.getResponseAppNotFound(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (AppInfoNotFoundException e) {
			return RestApiResponseHandler.getResponseAppInfoNotFound(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (ClientIDLoginFailedException e) {
			logger.error("login UAA with client ID " + e.getClientID() + " failed");
			return RestApiResponseHandler.getResponseInternalServerError(MessageUtil.getMessageString(
					MessageUtil.RestResponseErrorMsg_internal_server_error, LocaleUtil.getLocale(httpServletRequest)));
		}

		catch (Exception e) {
			logger.info("error in getOrgSpaceByAppId: " + e.getMessage());
			return RestApiResponseHandler.getResponseInternalServerError(
					new InternalServerErrorException(
							MessageUtil.RestResponseErrorMsg_retrieve_org_sapce_information_context, e),
					LocaleUtil.getLocale(httpServletRequest));
		}

		Map<String, String> check_result;
		try {
			check_result = ValidateUtil.handleInput(DataType.CREATE_REQUEST, jsonString, service_info,
					httpServletRequest);
			if (check_result.get("result") != "OK")
				return RestApiResponseHandler
						.getResponseBadRequest("{\"error\":" + check_result.get("error_message") + "\"}");
		} catch (InputJSONParseErrorException e) {
			return RestApiResponseHandler.getResponseInputJsonParseError(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (InputJSONFormatErrorException e) {
			return RestApiResponseHandler.getResponseInputJsonFormatError(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (Exception e) {
			return RestApiResponseHandler.getResponseInternalServerError(
					new InternalServerErrorException(MessageUtil.RestResponseErrorMsg_parse_input_json_context, e),
					LocaleUtil.getLocale(httpServletRequest));
		}

		String new_json = check_result.get("json");

		// build request to current auto-scaling server instance
		if ((policyId != null) && !(policyId.equals("null")) && (policyId.length() > 0)) // Update existing policy
		{
			logger.info("policyId: " + policyId + " already exist, now we update it");
			try {
				request_url = server_url + "/resources/policies/" + policyId;
				WebResource webResource = client.resource(request_url);
				String authorization = ConfigManager.getInternalAuthToken();
				ClientResponse response = webResource.header("Authorization", "Basic " + authorization)
						.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
						.put(ClientResponse.class, new_json);
				if (response.getStatus() == HttpServletResponse.SC_OK) // update OK
				{
					return RestApiResponseHandler.getResponseJsonOk("{\"policyId\":" + "\"" + policyId + "\"}");
				} else {
					String response_body = response.getEntity(String.class);
					if (response.getStatus() == HttpServletResponse.SC_BAD_REQUEST) {
						logger.info("Get back-end server bad request error  : " + response.getStatus()
								+ " with response body: " + response_body);
						return RestApiResponseHandler.getResponseInternalServerError(
								new InternalServerErrorException(
										MessageUtil.RestResponseErrorMsg_create_policy_in_Create_Policy_context),
								LocaleUtil.getLocale(httpServletRequest));
					} else {
						logger.info("Get back-end server error  : " + response.getStatus() + " with response body: "
								+ response_body);
						return RestApiResponseHandler.getResponseInternalServerError(
								new InternalServerErrorException(
										MessageUtil.RestResponseErrorMsg_create_policy_in_Create_Policy_context),
								LocaleUtil.getLocale(httpServletRequest));
					}
				}
			} catch (Exception e) {
				logger.info("error in update policy : " + e.getMessage());
				return RestApiResponseHandler.getResponseInternalServerError(
						new InternalServerErrorException(
								MessageUtil.RestResponseErrorMsg_create_policy_in_Create_Policy_context, e),
						LocaleUtil.getLocale(httpServletRequest));
			}
		} else { // Create new policy and attach
			logger.info("Create new policy and attach it with application");
			try {
				request_url = server_url + "/resources/policies?org=" + orgSpace.get(CloudFoundryManager.ORG_GUID)
						+ "&space=" + orgSpace.get(CloudFoundryManager.SPACE_GUID);
				WebResource webResource = client.resource(request_url);
				String authorization = ConfigManager.getInternalAuthToken();
				ClientResponse response = webResource.header("Authorization", "Basic " + authorization)
						.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
						.post(ClientResponse.class, new_json);
				String response_body = response.getEntity(String.class);
				JsonNode body_map = objectMapper.readTree(response_body);
				if (response.getStatus() == HttpServletResponse.SC_CREATED) {
					policyId = body_map.get("policyId").asText();
				} else {
					response_body = response.getEntity(String.class);
					if (response.getStatus() == HttpServletResponse.SC_BAD_REQUEST) {
						logger.info("Get back-end server bad request error  : " + response.getStatus()
								+ " with response body: " + response_body);
						return RestApiResponseHandler.getResponseInternalServerError(
								new InternalServerErrorException(
										MessageUtil.RestResponseErrorMsg_create_policy_in_Create_Policy_context),
								LocaleUtil.getLocale(httpServletRequest));
					} else {
						logger.info("Get back-end server error  : " + response.getStatus() + " with response body: "
								+ response_body);
						return RestApiResponseHandler.getResponseInternalServerError(
								new InternalServerErrorException(
										MessageUtil.RestResponseErrorMsg_create_policy_in_Create_Policy_context),
								LocaleUtil.getLocale(httpServletRequest));
					}
				}

				request_url = server_url + "/resources/apps/" + app_id;
				webResource = client.resource(request_url);
				String new_jsonstring = "{\"policyId\":" + "\"" + policyId + "\"," + "\"state\":" + "\"" + "enabled"
						+ "\"}";
				response = webResource.header("Authorization", "Basic " + authorization)
						.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
						.put(ClientResponse.class, new_jsonstring);
				response_body = response.getEntity(String.class);
				if (response.getStatus() == HttpServletResponse.SC_OK) // attach OK
				{
					return RestApiResponseHandler.getResponseCreatedOk("{\"policyId\":" + "\"" + policyId + "\"}");
				} else {
					response_body = response.getEntity(String.class);
					if (response.getStatus() == HttpServletResponse.SC_BAD_REQUEST) {
						logger.info("Get back-end server bad request error  : " + response.getStatus()
								+ " with response body: " + response_body);
						return RestApiResponseHandler.getResponseInternalServerError(
								new InternalServerErrorException(
										MessageUtil.RestResponseErrorMsg_create_policy_in_Create_Policy_context),
								LocaleUtil.getLocale(httpServletRequest));
					} else {
						logger.info("Get back-end server error  : " + response.getStatus() + " with response body: "
								+ response_body);
						return RestApiResponseHandler.getResponseInternalServerError(
								new InternalServerErrorException(
										MessageUtil.RestResponseErrorMsg_create_policy_in_Create_Policy_context),
								LocaleUtil.getLocale(httpServletRequest));
					}
				}
			} catch (Exception e) {
				logger.info("error in create policy and attach : " + e.getMessage());
				return RestApiResponseHandler.getResponseInternalServerError(
						new InternalServerErrorException(
								MessageUtil.RestResponseErrorMsg_create_policy_in_Create_Policy_context, e),
						LocaleUtil.getLocale(httpServletRequest));
			}
		}
	}

	@DELETE
	@Path("/{app_id}/policy")
	// @Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response deletePolicy(@Context final HttpServletRequest httpServletRequest,
			@PathParam("app_id") String app_id, String jsonString) {
		logger.info("Received JSON String of policy content: " + jsonString);

		Client client = RestUtil.getHTTPSRestClient();
		String policyId;
		String server_url;
		String request_url;
		try {
			Map<String, String> service_info = getserverinfo(app_id);
			server_url = service_info.get("server_url");
			policyId = service_info.get("policyId");
		} catch (CloudException e) {
			return RestApiResponseHandler.getResponseCloudError(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (AppNotFoundException e) {
			return RestApiResponseHandler.getResponseAppNotFound(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (ServiceNotFoundException e) {
			return RestApiResponseHandler.getResponseServiceNotFound(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (InternalAuthenticationException e) {
			return RestApiResponseHandler.getResponseInternalAuthenticationFail(e,
					LocaleUtil.getLocale(httpServletRequest));
		} catch (ClientIDLoginFailedException e) {
			logger.error("login UAA with client ID " + e.getClientID() + " failed");
			return RestApiResponseHandler.getResponseInternalServerError(MessageUtil.getMessageString(
					MessageUtil.RestResponseErrorMsg_internal_server_error, LocaleUtil.getLocale(httpServletRequest)));
		} catch (Exception e) {
			logger.info("error in getserverinfo: " + e.getMessage());
			return RestApiResponseHandler.getResponseInternalServerError(
					new InternalServerErrorException(
							MessageUtil.RestResponseErrorMsg_retrieve_application_service_information_context, e),
					LocaleUtil.getLocale(httpServletRequest));
		}

		if ((policyId != null) && !(policyId.equals("null")) && (policyId.length() > 0)) // try to detach the
																							// application and policy,
																							// then delete the policy
		{
			try {
				// detach application and policy
				request_url = server_url + "/resources/apps/" + app_id + "?policyId=" + policyId + "&state=disabled";
				WebResource webResource = client.resource(request_url);
				String authorization = ConfigManager.getInternalAuthToken();
				ClientResponse response = webResource.header("Authorization", "Basic " + authorization)
						.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
						.delete(ClientResponse.class);
				if (response.getStatus() != HttpServletResponse.SC_OK) {// update OK
					String response_body = response.getEntity(String.class);
					if (response.getStatus() == HttpServletResponse.SC_BAD_REQUEST) {
						logger.info("Get back-end server bad request error  : " + response.getStatus()
								+ " with response body: " + response_body);
						return RestApiResponseHandler.getResponseInternalServerError(
								new InternalServerErrorException(
										MessageUtil.RestResponseErrorMsg_detach_policy_in_Delete_Policy_context),
								LocaleUtil.getLocale(httpServletRequest));
					} else {
						logger.info("Get back-end server error  : " + response.getStatus() + " with response body: "
								+ response_body);
						return RestApiResponseHandler.getResponseInternalServerError(
								new InternalServerErrorException(
										MessageUtil.RestResponseErrorMsg_detach_policy_in_Delete_Policy_context),
								LocaleUtil.getLocale(httpServletRequest));
					}
				}

				// delete policy

				request_url = server_url + "/resources/policies/" + policyId;
				webResource = client.resource(request_url);
				response = webResource.header("Authorization", "Basic " + authorization).delete(ClientResponse.class);
				if (response.getStatus() == 204) // delete OK
				{
					return RestApiResponseHandler.getResponse200Ok("{}");
				} else {
					String response_body = response.getEntity(String.class);
					if (response.getStatus() == HttpServletResponse.SC_BAD_REQUEST) {
						logger.info("Get back-end server bad request error  : " + response.getStatus()
								+ " with response body: " + response_body);
						return RestApiResponseHandler.getResponseInternalServerError(
								new InternalServerErrorException(
										MessageUtil.RestResponseErrorMsg_delete_policy_in_Delete_Policy_context),
								LocaleUtil.getLocale(httpServletRequest));
					} else {
						logger.info("Get back-end server error  : " + response.getStatus() + " with response body: "
								+ response_body);
						return RestApiResponseHandler.getResponseInternalServerError(
								new InternalServerErrorException(
										MessageUtil.RestResponseErrorMsg_delete_policy_in_Delete_Policy_context),
								LocaleUtil.getLocale(httpServletRequest));
					}
				}
			} catch (Exception e) {
				logger.info("error in delete policy : " + e.getMessage());
				return RestApiResponseHandler.getResponseInternalServerError(
						new InternalServerErrorException(
								MessageUtil.RestResponseErrorMsg_delete_policy_in_Delete_Policy_context, e),
						LocaleUtil.getLocale(httpServletRequest));
			}

		} else { // Can not find policyId for this application
			return RestApiResponseHandler.getResponsePolicyNotExistError(new PolicyNotFoundException(app_id),
					LocaleUtil.getLocale(httpServletRequest));
		}

	}

	@GET
	@Path("/{app_id}/policy")
	// @Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response getPolicy(@Context final HttpServletRequest httpServletRequest,
			@PathParam("app_id") String app_id) {

		Client client = RestUtil.getHTTPSRestClient();
		String policyId;
		String server_url;
		String request_url;
		String status;
		Map<String, String> service_info;
		try {
			service_info = getserverinfo(app_id);
			server_url = service_info.get("server_url");
			policyId = service_info.get("policyId");
			status = service_info.get("status");
		} catch (CloudException e) {
			return RestApiResponseHandler.getResponseCloudError(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (AppNotFoundException e) {
			return RestApiResponseHandler.getResponseAppNotFound(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (ServiceNotFoundException e) {
			return RestApiResponseHandler.getResponseServiceNotFound(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (InternalAuthenticationException e) {
			return RestApiResponseHandler.getResponseInternalAuthenticationFail(e,
					LocaleUtil.getLocale(httpServletRequest));
		} catch (ClientIDLoginFailedException e) {
			logger.error("login UAA with client ID " + e.getClientID() + " failed");
			return RestApiResponseHandler.getResponseInternalServerError(MessageUtil.getMessageString(
					MessageUtil.RestResponseErrorMsg_internal_server_error, LocaleUtil.getLocale(httpServletRequest)));
		} catch (Exception e) {
			logger.info("error in getserverinfo: " + e.getMessage());
			return RestApiResponseHandler.getResponseInternalServerError(
					new InternalServerErrorException(
							MessageUtil.RestResponseErrorMsg_retrieve_application_service_information_context, e),
					LocaleUtil.getLocale(httpServletRequest));
		}

		// build request to current autoscaling server instance
		if ((policyId != null) && !(policyId.equals("null")) && (policyId.length() > 0)) // get policy information
		{
			try {

				// get policy information
				request_url = server_url + "/resources/policies/" + policyId;
				WebResource webResource = client.resource(request_url);
				String authorization = ConfigManager.getInternalAuthToken();
				ClientResponse response = webResource.header("Authorization", "Basic " + authorization)
						.accept(MediaType.APPLICATION_JSON).get(ClientResponse.class);
				if (response.getStatus() == HttpServletResponse.SC_OK) // get OK
				{
					Map<String, String> supplyment = new HashMap<String, String>();
					supplyment.put("policyState", status.toUpperCase());
					Map<String, String> check_result = ValidateUtil.handleOutput(DataType.GET_RESPONSE,
							(String) (response.getEntity(String.class)), supplyment, service_info, httpServletRequest);
					if (check_result.get("result") != "OK")
						return RestApiResponseHandler.getResponseBadRequest(check_result.get("error_message"));
					return RestApiResponseHandler.getResponseJsonOk(check_result.get("json"));
				} else {
					String response_body = response.getEntity(String.class);
					if (response.getStatus() == HttpServletResponse.SC_BAD_REQUEST) {
						logger.info("Get back-end server bad request error  : " + response.getStatus()
								+ " with response body: " + response_body);
						return RestApiResponseHandler.getResponseInternalServerError(
								new InternalServerErrorException(MessageUtil.RestResponseErrorMsg_Get_Policy_context),
								LocaleUtil.getLocale(httpServletRequest));
					} else {
						logger.info("Get back-end server error  : " + response.getStatus() + " with response body: "
								+ response_body);
						return RestApiResponseHandler.getResponseInternalServerError(
								new InternalServerErrorException(MessageUtil.RestResponseErrorMsg_Get_Policy_context),
								LocaleUtil.getLocale(httpServletRequest));
					}
				}

			} catch (OutputJSONParseErrorException e) {
				return RestApiResponseHandler.getResponseOutputJsonParseError(e,
						LocaleUtil.getLocale(httpServletRequest));
			} catch (OutputJSONFormatErrorException e) {
				return RestApiResponseHandler.getResponseOutputJsonFormatError(e,
						LocaleUtil.getLocale(httpServletRequest));
			} catch (Exception e) {
				logger.info("error in get policy : " + e.getMessage());
				return RestApiResponseHandler.getResponseInternalServerError(
						new InternalServerErrorException(MessageUtil.RestResponseErrorMsg_Get_Policy_context, e),
						LocaleUtil.getLocale(httpServletRequest));
			}

		} else {
			return RestApiResponseHandler.getResponsePolicyNotExistError(new PolicyNotFoundException(app_id),
					LocaleUtil.getLocale(httpServletRequest));
		}
	}

	@PUT
	@Path("/{app_id}/policy/status")
	@Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response enablePolicy(@Context final HttpServletRequest httpServletRequest,
			@PathParam("app_id") String app_id, String jsonString) {
		logger.info("Received JSON String of policy content: " + jsonString);

		Client client = RestUtil.getHTTPSRestClient();
		String policyId;
		String server_url;
		String request_url;
		Map<String, String> service_info;
		try {
			service_info = getserverinfo(app_id);
			server_url = service_info.get("server_url");
			policyId = service_info.get("policyId");
		} catch (CloudException e) {
			return RestApiResponseHandler.getResponseCloudError(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (AppNotFoundException e) {
			return RestApiResponseHandler.getResponseAppNotFound(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (ServiceNotFoundException e) {
			return RestApiResponseHandler.getResponseServiceNotFound(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (InternalAuthenticationException e) {
			return RestApiResponseHandler.getResponseInternalAuthenticationFail(e,
					LocaleUtil.getLocale(httpServletRequest));
		} catch (ClientIDLoginFailedException e) {
			logger.error("login UAA with client ID " + e.getClientID() + " failed");
			return RestApiResponseHandler.getResponseInternalServerError(MessageUtil.getMessageString(
					MessageUtil.RestResponseErrorMsg_internal_server_error, LocaleUtil.getLocale(httpServletRequest)));
		} catch (Exception e) {
			logger.info("error in getserverinfo: " + e.getMessage());
			return RestApiResponseHandler.getResponseInternalServerError(
					new InternalServerErrorException(
							MessageUtil.RestResponseErrorMsg_retrieve_application_service_information_context, e),
					LocaleUtil.getLocale(httpServletRequest));
		}

		// validate and transform the input
		Map<String, String> check_result;
		try {
			check_result = ValidateUtil.handleInput(DataType.ENABLE_REQUEST, jsonString, service_info,
					httpServletRequest);
			if (check_result.get("result") != "OK")
				return RestApiResponseHandler.getResponseBadRequest(check_result.get("error_message"));
		} catch (InputJSONParseErrorException e) {
			return RestApiResponseHandler.getResponseInputJsonParseError(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (InputJSONFormatErrorException e) {
			return RestApiResponseHandler.getResponseInputJsonFormatError(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (Exception e) {
			return RestApiResponseHandler.getResponseInternalServerError(
					new InternalServerErrorException(MessageUtil.RestResponseErrorMsg_parse_input_json_context, e),
					LocaleUtil.getLocale(httpServletRequest));
		}

		String new_json = check_result.get("json");

		// build request to current autos－scaling server instance
		if ((policyId != null) && !(policyId.equals("null")) && (policyId.length() > 0)) // get policy information
		{
			try {
				// update policy status, the same as attach
				request_url = server_url + "/resources/apps/" + app_id;
				WebResource webResource = client.resource(request_url);
				// debug should call handleInput2()
				JSONObject json = new JSONObject(new_json);
				json.put("policyId", policyId);
				String authorization = ConfigManager.getInternalAuthToken();
				ClientResponse response = webResource.header("Authorization", "Basic " + authorization)
						.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
						.put(ClientResponse.class, json.toString());
				if (response.getStatus() == HttpServletResponse.SC_OK) // enable OK
				{
					return RestApiResponseHandler.getResponseJsonOk("{}");
				} else {
					String response_body = response.getEntity(String.class);
					if (response.getStatus() == HttpServletResponse.SC_BAD_REQUEST) {
						logger.info("Get back-end server bad request error  : " + response.getStatus()
								+ " with response body: " + response_body);
						return RestApiResponseHandler.getResponseInternalServerError(
								new InternalServerErrorException(
										MessageUtil.RestResponseErrorMsg_Enable_Policy_context),
								LocaleUtil.getLocale(httpServletRequest));
					} else {
						logger.info("Get back-end server error  : " + response.getStatus() + " with response body: "
								+ response_body);
						return RestApiResponseHandler.getResponseInternalServerError(
								new InternalServerErrorException(
										MessageUtil.RestResponseErrorMsg_Enable_Policy_context),
								LocaleUtil.getLocale(httpServletRequest));
					}
				}

			} catch (Exception e) {
				logger.info("error in enable policy : " + e.getMessage());
				return RestApiResponseHandler.getResponseInternalServerError(
						new InternalServerErrorException(MessageUtil.RestResponseErrorMsg_Enable_Policy_context, e),
						LocaleUtil.getLocale(httpServletRequest));
			}

		} else {
			return RestApiResponseHandler.getResponsePolicyNotExistError(new PolicyNotFoundException(app_id),
					LocaleUtil.getLocale(httpServletRequest));
		}

	}

	@GET
	@Path("/{app_id}/policy/status")
	// @Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response getPolicyStatus(@Context final HttpServletRequest httpServletRequest,
			@PathParam("app_id") String app_id) {

		String policyStatus;
		try {
			Map<String, String> service_info = getserverinfo(app_id);
			policyStatus = service_info.get("status");
			if ((policyStatus != null) && (policyStatus.length() > 0)) {
				return RestApiResponseHandler
						.getResponseJsonOk("{\"status\":" + "\"" + policyStatus.toUpperCase() + "\"}");
			} else {
				return RestApiResponseHandler.getResponsePolicyNotExistError(new PolicyNotFoundException(app_id),
						LocaleUtil.getLocale(httpServletRequest));
			}
		} catch (CloudException e) {
			return RestApiResponseHandler.getResponseCloudError(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (AppNotFoundException e) {
			return RestApiResponseHandler.getResponseAppNotFound(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (ServiceNotFoundException e) {
			return RestApiResponseHandler.getResponseServiceNotFound(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (InternalAuthenticationException e) {
			return RestApiResponseHandler.getResponseInternalAuthenticationFail(e,
					LocaleUtil.getLocale(httpServletRequest));
		} catch (ClientIDLoginFailedException e) {
			logger.error("login UAA with client ID " + e.getClientID() + " failed");
			return RestApiResponseHandler.getResponseInternalServerError(MessageUtil.getMessageString(
					MessageUtil.RestResponseErrorMsg_internal_server_error, LocaleUtil.getLocale(httpServletRequest)));
		} catch (Exception e) {
			logger.info("error in getserverinfo: " + e.getMessage());
			return RestApiResponseHandler.getResponseInternalServerError(
					new InternalServerErrorException(
							MessageUtil.RestResponseErrorMsg_retrieve_application_service_information_context, e),
					LocaleUtil.getLocale(httpServletRequest));
		}
	}

	@GET
	@Path("/{app_id}/scalinghistory")
	// @Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response getHistory(@Context final HttpServletRequest httpServletRequest, @PathParam("app_id") String app_id,
			@QueryParam("startTime") String startTime, @QueryParam("endTime") String endTime) {

		Client client = RestUtil.getHTTPSRestClient();
		String server_url;
		String request_url;
		Map<String, String> service_info;
		try {
			service_info = getserverinfo(app_id);
			server_url = service_info.get("server_url");
		} catch (CloudException e) {
			return RestApiResponseHandler.getResponseCloudError(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (AppNotFoundException e) {
			return RestApiResponseHandler.getResponseAppNotFound(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (ServiceNotFoundException e) {
			return RestApiResponseHandler.getResponseServiceNotFound(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (InternalAuthenticationException e) {
			return RestApiResponseHandler.getResponseInternalAuthenticationFail(e,
					LocaleUtil.getLocale(httpServletRequest));
		} catch (ClientIDLoginFailedException e) {
			logger.error("login UAA with client ID " + e.getClientID() + " failed");
			return RestApiResponseHandler.getResponseInternalServerError(MessageUtil.getMessageString(
					MessageUtil.RestResponseErrorMsg_internal_server_error, LocaleUtil.getLocale(httpServletRequest)));
		} catch (Exception e) {
			logger.info("error in getserverinfo: " + e.getMessage());
			return RestApiResponseHandler.getResponseInternalServerError(
					new InternalServerErrorException(
							MessageUtil.RestResponseErrorMsg_retrieve_application_service_information_context, e),
					LocaleUtil.getLocale(httpServletRequest));
		}

		// validate and transform the input

		try {
			// get scaling history information
			long now = new Date().getTime();
			long new_start_time = 0, new_end_time = 0;
			if (!ValidateUtil.isNull(startTime)) {
				new_start_time = Long.parseLong(startTime);
			}

			if (!ValidateUtil.isNull(endTime)) {
				new_end_time = Long.parseLong(endTime);
			}

			// TimeRange is in form of minutes
			if ((new_start_time <= 0) && (new_end_time <= 0)) { // both startTime and endTime are not specified or wrong
				new_start_time = now - Constants.DASHBORAD_TIME_RANGE * 60 * 1000 * 100; // debug
				new_end_time = now;
			} else if ((new_start_time > 0) && (new_end_time <= 0)) {
				new_end_time = new_start_time + Constants.DASHBORAD_TIME_RANGE * 60 * 1000 * 100;
			} else if ((new_start_time <= 0) && (new_end_time > 0)) {
				new_start_time = new_end_time - Constants.DASHBORAD_TIME_RANGE * 60 * 1000 * 100;
				if (new_start_time < 0)
					new_start_time = 0;
			}

			long time_range = new_end_time - new_start_time;
			if (time_range <= 0) { // whatever new_end_time <= 0(endTime now specified or wrong) or not (endTime
									// specified)
				if (new_start_time < now) {
					new_end_time = now;
				} else {
					new_start_time = now - Constants.DASHBORAD_TIME_RANGE * 60 * 1000 * 100; // debug
					new_end_time = now;
				}
			}

			request_url = server_url + "/resources/history/" + "?appId=" + app_id + "&startTime=" + new_start_time
					+ "&endTime=" + new_end_time;
			WebResource webResource = client.resource(request_url);
			String authorization = ConfigManager.getInternalAuthToken();
			ClientResponse response = webResource.header("Authorization", "Basic " + authorization)
					.accept(MediaType.APPLICATION_JSON).get(ClientResponse.class);
			if (response.getStatus() == HttpServletResponse.SC_OK) // get OK
			{
				String result = response.getEntity(String.class);
				Map<String, String> check_result = ValidateUtil.handleOutput(DataType.GET_HISTORY_RESPONSE, result,
						new HashMap<String, String>(), service_info, httpServletRequest);
				if (check_result.get("result") != "OK")
					return RestApiResponseHandler.getResponseBadRequest(check_result.get("error_message"));
				return RestApiResponseHandler.getResponseJsonOk(check_result.get("json"));
			} else {
				String response_body = response.getEntity(String.class);
				if (response.getStatus() == HttpServletResponse.SC_BAD_REQUEST) {
					logger.info("Get back-end server bad request error  : " + response.getStatus()
							+ " with response body: " + response_body);
					return RestApiResponseHandler.getResponseInternalServerError(
							new InternalServerErrorException(
									MessageUtil.RestResponseErrorMsg_Get_Scaling_History_context),
							LocaleUtil.getLocale(httpServletRequest));
				} else {
					logger.info("Get back-end server error  : " + response.getStatus() + " with response body: "
							+ response_body);
					return RestApiResponseHandler.getResponseInternalServerError(
							new InternalServerErrorException(
									MessageUtil.RestResponseErrorMsg_Get_Scaling_History_context),
							LocaleUtil.getLocale(httpServletRequest));
				}
			}

		} catch (OutputJSONParseErrorException e) {
			return RestApiResponseHandler.getResponseOutputJsonParseError(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (OutputJSONFormatErrorException e) {
			return RestApiResponseHandler.getResponseOutputJsonFormatError(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (Exception e) {
			logger.info("error in get history : " + e.getMessage());
			return RestApiResponseHandler.getResponseInternalServerError(
					new InternalServerErrorException(MessageUtil.RestResponseErrorMsg_Get_Scaling_History_context, e),
					LocaleUtil.getLocale(httpServletRequest));
		}

	}

	@GET
	@Path("/{app_id}/metrics")
	// @Consumes(MediaType.APPLICATION_JSON)
	@Produces(MediaType.APPLICATION_JSON)
	public Response getMetrics(@Context final HttpServletRequest httpServletRequest, @PathParam("app_id") String app_id,
			@QueryParam("startTime") String startTime, @QueryParam("endTime") String endTime) {

		Client client = RestUtil.getHTTPSRestClient();
		String server_url;
		String service_id;
		String request_url;
		Map<String, String> service_info;
		try {
			service_info = getserverinfo(app_id);
			server_url = service_info.get("server_url");
			service_id = service_info.get("service_id");
		} catch (CloudException e) {
			return RestApiResponseHandler.getResponseCloudError(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (AppNotFoundException e) {
			return RestApiResponseHandler.getResponseAppNotFound(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (ServiceNotFoundException e) {
			return RestApiResponseHandler.getResponseServiceNotFound(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (InternalAuthenticationException e) {
			return RestApiResponseHandler.getResponseInternalAuthenticationFail(e,
					LocaleUtil.getLocale(httpServletRequest));
		} catch (ClientIDLoginFailedException e) {
			logger.error("login UAA with client ID " + e.getClientID() + " failed");
			return RestApiResponseHandler.getResponseInternalServerError(MessageUtil.getMessageString(
					MessageUtil.RestResponseErrorMsg_internal_server_error, LocaleUtil.getLocale(httpServletRequest)));
		} catch (Exception e) {
			logger.info("error in getserverinfo: " + e.getMessage());
			return RestApiResponseHandler.getResponseInternalServerError(
					new InternalServerErrorException(
							MessageUtil.RestResponseErrorMsg_retrieve_application_service_information_context, e),
					LocaleUtil.getLocale(httpServletRequest));
		}

		try {
			long newerThan = 0, timeRange = 0;

			if (!ValidateUtil.isNull(startTime)) {
				newerThan = Long.parseLong(startTime);
			}

			if (!ValidateUtil.isNull(endTime)) {
				timeRange = (Long.parseLong(endTime) - newerThan) / (60 * 1000);
			}
			long now = new Date().getTime();

			// TimeRange is in form of minutes
			if (timeRange <= 0) // not specify endTime (==0) or endTime is earlier than startTime (<0)
				timeRange = Constants.DASHBORAD_TIME_RANGE;

			if (newerThan == 0) {
				newerThan = now - timeRange * 60 * 1000;
				// when startTime not specify and endtime is later than now, newerThan might be negative
				if (newerThan <= 0)
					newerThan = 0;
			}

			request_url = server_url + "/services/metrics/" + service_id + "/" + app_id + "?newerThan=" + newerThan
					+ "&timeRange=" + timeRange;
			WebResource webResource = client.resource(request_url);
			String authorization = ConfigManager.getInternalAuthToken();
			ClientResponse response = webResource.header("Authorization", "Basic " + authorization)
					.accept(MediaType.APPLICATION_JSON).get(ClientResponse.class);
			if (response.getStatus() == HttpServletResponse.SC_OK) // get OK
			{
				String result = response.getEntity(String.class);
				Map<String, String> check_result = ValidateUtil.handleOutput(DataType.GET_METRICS_RESPONSE, result,
						new HashMap<String, String>(), service_info, httpServletRequest);
				if (check_result.get("result") != "OK")
					return RestApiResponseHandler.getResponseBadRequest(check_result.get("error_message"));
				return RestApiResponseHandler.getResponseJsonOk(check_result.get("json"));
			} else {
				String response_body = response.getEntity(String.class);
				if (response.getStatus() == HttpServletResponse.SC_BAD_REQUEST) {
					logger.info("Get back-end server bad request error  : " + response.getStatus()
							+ " with response body: " + response_body);
					return RestApiResponseHandler.getResponseInternalServerError(
							new InternalServerErrorException(MessageUtil.RestResponseErrorMsg_Get_Metric_Data_context),
							LocaleUtil.getLocale(httpServletRequest));
				} else {
					logger.info("Get back-end server error  : " + response.getStatus() + " with response body: "
							+ response_body);
					return RestApiResponseHandler.getResponseInternalServerError(
							new InternalServerErrorException(MessageUtil.RestResponseErrorMsg_Get_Metric_Data_context),
							LocaleUtil.getLocale(httpServletRequest));
				}
			}

		} catch (OutputJSONParseErrorException e) {
			return RestApiResponseHandler.getResponseOutputJsonParseError(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (OutputJSONFormatErrorException e) {
			return RestApiResponseHandler.getResponseOutputJsonFormatError(e, LocaleUtil.getLocale(httpServletRequest));
		} catch (Exception e) {
			logger.info("error in get metric : " + e.getMessage());
			return RestApiResponseHandler.getResponseInternalServerError(
					new InternalServerErrorException(MessageUtil.RestResponseErrorMsg_Get_Metric_Data_context, e),
					LocaleUtil.getLocale(httpServletRequest));
		}

	}

	public Map<String, String> getserverinfo(String app_id) throws Exception {
		Map<String, String> serviceInfo = new HashMap<String, String>();

		try {
			String service_name = ConfigManager.get("scalingServiceName", "CF-AutoScaler");
			JsonNode service_info = CloudFoundryManager.getInstance().getServiceInfo(app_id, service_name);
			JsonNode credentials = service_info.get("credentials");
			String request_url = credentials.get("url").asText() + "/resources/apps/" + app_id + "?serviceid="
					+ credentials.get("service_id").asText();
			logger.info("request_url: " + request_url);
			Client client = RestUtil.getHTTPSRestClient();
			WebResource webResource = client.resource(request_url);
			String authorization = ConfigManager.getInternalAuthToken();
			ClientResponse cr = webResource.header("Authorization", "Basic " + authorization)
					.accept(MediaType.APPLICATION_JSON).get(ClientResponse.class);
			int status_code = cr.getStatus();
			logger.info(">>>>>> Got status_code: " + status_code + " from Server in getserverinfo ");
			if (status_code != HttpServletResponse.SC_OK)
				if (status_code == HttpServletResponse.SC_UNAUTHORIZED)
					throw new InternalAuthenticationException("get Application information");
				else
					throw new CloudException();
			String response = cr.getEntity(String.class);
			logger.info(">>>" + response);
			JsonNode jobj = objectMapper.readTree(response);
			serviceInfo.put("service_id", credentials.get("service_id").asText());
			serviceInfo.put("server_url", credentials.get("url").asText());
			serviceInfo.put("policyId", jobj.get("policyId").asText());
			serviceInfo.put("appType", jobj.get("type").asText());
			serviceInfo.put("status", jobj.get("state").asText());

			return serviceInfo;

		} catch (InternalAuthenticationException e) {
			throw e;
		} catch (IOException e) {
			throw new CloudException(e);
		} catch (CloudException e) {
			throw new CloudException(e);
		} catch (AppNotFoundException e) {
			throw new AppNotFoundException(e.getAppId(), e);
		} catch (ServiceNotFoundException e) {
			throw new ServiceNotFoundException(e.getServiceName(), e.getAppId(), e);
		} catch (ClientIDLoginFailedException e) {
			throw e;
		}
	}
}
