package org.cloudfoundry.autoscaler.common.util;

import java.io.IOException;
import java.util.Base64;
import java.util.HashMap;
import java.util.Map;
import java.util.Set;

import javax.ws.rs.core.MediaType;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.common.AppInfoNotFoundException;
import org.cloudfoundry.autoscaler.common.AppNotFoundException;
import org.cloudfoundry.autoscaler.common.ClientIDLoginFailedException;
import org.cloudfoundry.autoscaler.common.CloudException;
import org.cloudfoundry.autoscaler.common.Constants;
import org.cloudfoundry.autoscaler.common.ServiceNotFoundException;
import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.sun.jersey.api.client.Client;
import com.sun.jersey.api.client.ClientResponse;
import com.sun.jersey.api.client.WebResource;
import com.sun.jersey.api.representation.Form;

public class CloudFoundryManager {
	private static final Logger logger = Logger.getLogger(CloudFoundryManager.class);
	public static final String ORG_NAME = "ORG_NAME";
	public static final String ORG_GUID = "ORG_GUID";
	public static final String SPACE_NAME = "SPACE_NAME";
	public static final String SPACE_GUID = "SPACE_GUID";
	public static final String APP_GUID = "APP_GUID";

	private String target;
	private String accessToken;
	private long accessTokenExpireTime;
	private String cfClientId;
	private String cfSecretKey;
	private Client restClient;

	private static String[][] appTypeMapper = { { Constants.APP_TYPE_JAVA, "(?i).*Liberty.*" },
			{ Constants.APP_TYPE_RUBY_ON_RAILS, "(?i).*Ruby/Rails.*" },
			{ Constants.APP_TYPE_RUBY_SINATRA, "(?i).*Ruby/Rack.*" }, { Constants.APP_TYPE_RUBY, "(?i).*Ruby.*" },
			{ Constants.APP_TYPE_NODEJS, "(?i).*(Node\\.js|nodejs).*" }, { Constants.APP_TYPE_GO, "(?i).*go.*" },
			{ Constants.APP_TYPE_PHP, "(?i).*php.*" }, { Constants.APP_TYPE_PYTHON, "(?i).*python.*" },
			{ Constants.APP_TYPE_DOTNET, "(?i).*dotnet.*" }, };

	private static volatile CloudFoundryManager instance;

	public CloudFoundryManager() {
		this(getClientId(), getSecretKey(), getCFAPIUrl());
	}

	public CloudFoundryManager(String clientId, String clientSecret, String cfUrl) {
		this.cfClientId = clientId;
		this.cfSecretKey = clientSecret;
		this.restClient = RestUtil.getHTTPSRestClient();
		if (!cfUrl.startsWith("http://") || cfUrl.startsWith("https://")) {
			this.target = "https://" + cfUrl;
		} else {
			this.target = cfUrl;
		}
	}

	public static CloudFoundryManager getInstance() throws Exception {
		if (instance == null) {
			synchronized (CloudFoundryManager.class) {
				if (instance == null)
					instance = new CloudFoundryManager();
			}
		}
		instance.login();
		return instance;
	}

	public void login() throws Exception {
		long now = System.currentTimeMillis();
		// log in again if the accessToken expires
		if (accessToken == null || now >= this.accessTokenExpireTime - 5 * 60 * 1000) {
			loginWithClientId();
		}
	}

	private void loginWithClientId() throws Exception {
		String infoUrl = target + "/info";
		logger.debug("connecting to URL:" + infoUrl);
		WebResource webResource = restClient.resource(infoUrl);
		String response = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
				.get(String.class);
		logger.debug(">>>" + response);

		String authString = cfClientId + ":" + cfSecretKey;
		String authStringEnc = Base64.getEncoder().encodeToString(authString.getBytes());
		JsonNode jobj = new ObjectMapper().readTree(response);
		String authorization_endpoint = jobj.get("authorization_endpoint").asText();

		logger.debug(">>>" + authorization_endpoint);
		webResource = restClient.resource(authorization_endpoint + "/oauth/token");
		ClientResponse cr = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_FORM_URLENCODED)
				.header("charset", "utf-8").header("authorization", "Basic " + authStringEnc).post(ClientResponse.class,
						"grant_type=client_credentials&client_id=" + cfClientId + "&client_secret=" + cfSecretKey);

		if (cr.getStatus() == 200) {
			response = cr.getEntity(String.class);
			logger.debug(">>>" + response);

			jobj = new ObjectMapper().readTree(response);

			accessToken = jobj.get("access_token").asText();
			long expire_in = Long.parseLong(jobj.get("expires_in").toString());
			accessTokenExpireTime = System.currentTimeMillis() + expire_in * 1000;

			logger.debug(">>>" + accessToken);
			return;
		} else {
			throw new ClientIDLoginFailedException(cfClientId, cr.getStatusInfo().toString());
		}

	}

	public JsonNode getServiceInfo(String appId, String serviceName) throws Exception {
		try {
			JsonNode appEnvJsonMap = this.getApplicationEnvByAppId(appId);
			logger.debug("appEnvJsonMap:" + appEnvJsonMap.toString());
			JsonNode sys_env = appEnvJsonMap.get("system_env_json");
			logger.debug("sys_env:" + sys_env.toString());
			JsonNode application_env = appEnvJsonMap.get("application_env_json");
			logger.debug("application_env:" + application_env.toString());
			JsonNode vcap_application = application_env.get("VCAP_APPLICATION");
			logger.debug("vcap_application:" + vcap_application.toString());
			String application_name = vcap_application.get("application_name").asText();
			logger.debug("application_name:" + application_name);

			JsonNode vcap_service = sys_env.get("VCAP_SERVICES");
			logger.debug("vcap_service:" + vcap_service.toString());
			JSONObject vcap_service_jobj = new JSONObject(vcap_service.toString());
			JSONArray services = (JSONArray) vcap_service_jobj.get(serviceName);
			JSONObject service_map_jobj = services.getJSONObject(0);
			JsonNode service_map = new ObjectMapper().readTree(service_map_jobj.toString());
			return service_map;

		} catch (CloudException e) {
			throw new CloudException(e);
		} catch (AppNotFoundException e) {
			throw new AppNotFoundException(e.getAppId(), e);
		} catch (IndexOutOfBoundsException e) {
			throw new ServiceNotFoundException(serviceName, appId, e);
		} catch (NullPointerException e) {
			throw new ServiceNotFoundException(serviceName, appId, e);
		} catch (Exception e) {
			throw new Exception("error: failed to get service information from VCAP_SERVICE for appId " + appId, e);
		}
	}

	public Map<String, String> findOrgSpaceForApp(String appName, String appVersion) throws Exception {

		String appGuid = null;

		// Query to find matching apps (with same name, but may belong to other
		// spaces/orgs, then match on version)
		try {
			String url = this.target + "/v2/apps?q=name:" + appName;
			logger.info("connecting to URL:" + url);
			WebResource webResource = restClient.resource(url);
			String response = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
					.header("Authorization", "Bearer " + this.accessToken).get(String.class);
			logger.info(">>>" + response);

			JSONObject jobj = new JSONObject(response);
			JSONArray jarray = (JSONArray) jobj.get("resources");
			appGuid = null;
			for (Object obj : jarray) {
				String version = (String) ((JSONObject) ((JSONObject) obj).get("entity")).get("version");
				logger.info(">>>> " + version);
				if (appVersion.equals(version)) {
					appGuid = (String) ((JSONObject) ((JSONObject) obj).get("metadata")).get("guid");
					break;
				}
			}
			if (appGuid == null) {
				throw new Exception("Could not find GUID for app: " + appName);
			}
		} catch (Exception e) {
			throw new Exception("appId=" + appName + " " + e.getMessage());
		}

		return getOrgSpaceByAppId(appGuid);
	}

	public Map<String, String> getOrgSpaceByAppId(String appId) throws Exception {

		Map<String, String> result = null;
		String spaceName = null;
		String spaceGuid = null;

		// find matching space for App GUID
		try {
			String url = this.target + "/v2/spaces?q=app_guid:" + appId;
			logger.info("connecting to URL:" + url);
			WebResource webResource = restClient.resource(url);
			ClientResponse cr = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
					.header("Authorization", "Bearer " + this.accessToken).get(ClientResponse.class);
			if (cr.getStatus() == 404) { // 404 will never be returned
				throw new AppNotFoundException(appId);
			}
			String response = cr.getEntity(String.class);
			logger.debug(">>>" + response);
			JSONObject jobj = new JSONObject(response);
			JSONArray jarray = (JSONArray) jobj.get("resources");
			if (jarray.length() == 0) {
				throw new AppNotFoundException(appId, "Could not find matching space for app");
			}
			if (jarray.length() != 1) {
				throw new AppInfoNotFoundException(appId, "Could not find specfic matching space for app");
			}
			spaceGuid = (String) ((JSONObject) ((JSONObject) jarray.get(0)).get("metadata")).get("guid");
			spaceName = (String) ((JSONObject) ((JSONObject) jarray.get(0)).get("entity")).get("name");
			if (spaceGuid == null || spaceName == null) {
				throw new AppInfoNotFoundException(appId, "Could not find  space for app");
			}
		} catch (AppNotFoundException e) {
			throw new AppNotFoundException(e.getAppId(), e.getMessage());
		} catch (AppInfoNotFoundException e) {
			throw new AppInfoNotFoundException(e.getAppId(), e.getMessage());
		} catch (Exception e) {
			throw new Exception("appId=" + appId + " " + e.getMessage());
		}

		// finally find the org name to which the space belongs
		try {
			String url = this.target + "/v2/organizations?q=space_guid:" + spaceGuid;
			logger.debug("connecting to URL:" + url);
			WebResource webResource = restClient.resource(url);
			ClientResponse cr = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
					.header("Authorization", "Bearer " + this.accessToken).get(ClientResponse.class);
			if (cr.getStatus() == 404) { // 404 will never be returned
				throw new AppNotFoundException(appId);
			}
			String response = cr.getEntity(String.class);
			logger.debug(">>>" + response);
			JSONObject jobj = new JSONObject(response);
			JSONArray jarray = (JSONArray) jobj.get("resources");
			if (jarray.length() == 0) {
				throw new AppNotFoundException(appId, "Could not find matching organization for app");
			}
			if (jarray.length() != 1) {
				throw new AppInfoNotFoundException(appId, "Could not find specfic matching organization for app");
			}
			String orgGuid = (String) ((JSONObject) ((JSONObject) jarray.get(0)).get("metadata")).get("guid");
			String orgName = (String) ((JSONObject) ((JSONObject) jarray.get(0)).get("entity")).get("name");
			if (orgName == null || orgGuid == null) {
				// throw new Exception("Could not find organization for app: " +
				// appId);
				throw new AppInfoNotFoundException(appId, "Could not find organization for app");
			}
			logger.info(">>> Found org_name =" + orgName + " space_name=" + spaceName + " for app " + appId);
			result = new HashMap<String, String>();
			result.put(SPACE_GUID, spaceGuid);
			result.put(SPACE_NAME, spaceName);
			result.put(ORG_GUID, orgGuid);
			result.put(ORG_NAME, orgName);
			result.put(APP_GUID, appId);
		} catch (AppNotFoundException e) {
			throw new AppNotFoundException(e.getAppId(), e.getMessage());
		} catch (AppInfoNotFoundException e) {
			throw new AppInfoNotFoundException(e.getAppId(), e.getMessage());
		} catch (Exception e) {
			throw new Exception("appId=" + appId + " " + e.getMessage());
		}

		return result;
	}

	public boolean check_token(String token) throws Exception {
		String authorization_endpoint = this.getUAAendpoint();
		logger.info(">>>" + authorization_endpoint);
		WebResource webResource = restClient.resource(authorization_endpoint + "/check_token");
		String authString = cfClientId + ":" + cfSecretKey;
		String authStringEnc = Base64.getEncoder().encodeToString(authString.getBytes());
		Form form = new Form();
		form.add("token", token);
		ClientResponse cr = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_FORM_URLENCODED)
				.header("charset", "utf-8").header("authorization", "Basic " + authStringEnc)
				.post(ClientResponse.class, form);

		String response = cr.getEntity(String.class);
		logger.info(">>>" + response);
		int status_code = cr.getStatus();
		return (status_code == 200);

	}

	public String getUAAendpoint() throws Exception {
		JsonNode InfoMap = this.getCfInfo();
		return InfoMap.get("authorization_endpoint").asText();
	}

	private JsonNode getCfInfo() throws Exception {
		String infoUrl = target + "/info";
		logger.debug("connecting to URL:" + infoUrl);
		WebResource webResource = restClient.resource(infoUrl);
		String response = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
				.get(String.class);
		logger.debug(">>>" + response);

		return new ObjectMapper().readTree(response);

	}

	private JsonNode getApplicationEnvByAppId(String appId) throws Exception {
		try {
			String url = this.target + "/v2/apps/" + appId + "/env";
			logger.debug("url:" + url);
			WebResource webResource = restClient.resource(url);

			ClientResponse cr = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
					.header("Authorization", "Bearer " + this.accessToken).get(ClientResponse.class);
			if (cr.getStatus() == 404) {
				throw new AppNotFoundException(appId);
			}
			String response = cr.getEntity(String.class);
			return new ObjectMapper().readTree(response);
		} catch (IOException e) {
			throw new CloudException(e);
		}
	}

	private static String getCFAPIUrl() {

		String key = Constants.CFURL;
		String cfUrl = System.getenv(key);

		if ((cfUrl == null) || cfUrl.isEmpty()) {
			try {
				String ApplicationEnvString = System.getenv("VCAP_APPLICATION");
				if (ApplicationEnvString != null) {
					JSONObject applicationEnv = new JSONObject(ApplicationEnvString);
					JSONArray applicationUris = (JSONArray) applicationEnv.get("application_uris");
					if (applicationUris.length() > 0) {
						String applicationUri = (String) applicationUris.get(0);
						cfUrl = "api." + applicationUri.substring(applicationUri.indexOf(".") + 1).trim();
					}
				}
			} catch (JSONException e) {
				logger.error(e.getMessage(), e);
			}
		}

		if ((cfUrl == null) || cfUrl.isEmpty()) {
			cfUrl = ConfigManager.get(key);
		}

		return cfUrl;

	}

	private static String getClientId() {
		return ConfigManager.get(Constants.CLIENT_ID);
	}

	private static String getSecretKey() {
		return ConfigManager.get(Constants.CLIENT_SECRET);
	}

	public String getAppIdByOrgSpaceAppName(String org, String space, String appName) throws Exception {
		String url = this.target + "/v2/organizations?q=name:" + org;
		logger.debug("url:" + url);
		WebResource webResource = restClient.resource(url);
		String response = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
				.header("Authorization", "Bearer " + this.accessToken).get(String.class);
		String orgId = getIdFromJson(response, org);
		if (orgId == null) {
			throw new Exception("Organization " + org + " does not exist.");
		}

		url = this.target + "/v2/organizations/" + orgId + "/spaces";
		webResource = restClient.resource(url);
		response = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
				.header("Authorization", "Bearer " + this.accessToken).get(String.class);
		String spaceId = getIdFromJson(response, space);
		if (spaceId == null) {
			throw new Exception("Space " + space + " does not exist in " + org + ".");
		}
		url = this.target + "/v2/spaces/" + spaceId + "/apps";
		webResource = restClient.resource(url);
		response = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
				.header("Authorization", "Bearer " + this.accessToken).get(String.class);
		String appId = getIdFromJson(response, appName);
		return appId;
	}

	private String getIdFromJson(String json, String name) throws Exception {
		String id = null;
		JSONObject jsonMap = new JSONObject(json);
		JSONArray records = (JSONArray) jsonMap.get("resources");
		for (int index = 0; index < records.length(); index++) {
			id = (String) ((JSONObject) ((JSONObject) records.get(index)).get("metadata")).get("guid");
			String Name = (String) ((JSONObject) ((JSONObject) records.get(index)).get("entity")).get("name");
			if (name.equals(Name)) {
				break;
			}
		}
		return id;
	}

	public String getAppType(String appId) throws Exception {
		return getAppInfoByAppId(appId)[1];
	}

	public String getAppNameByAppId(String appId) throws Exception {
		return getAppInfoByAppId(appId)[0];
	}

	public String[] getAppNameAndType(String appId) throws Exception {
		String[] appInfo = getAppInfoByAppId(appId);
		return new String[] { appInfo[0], appInfo[1] };
	}

	public String getAppStateByAppId(String appId) throws Exception {
		return getAppInfoByAppId(appId)[3];
	}

	public String[] getAppInfoByAppId(String appId) throws Exception {
		Map appJsonMap = this.getApplicationByAppId(appId);
		Map entity = (Map) appJsonMap.get("entity");
		String detectedBuildpack = (String) entity.get("detected_buildpack");
		if (detectedBuildpack == null) {
			detectedBuildpack = (String) entity.get("buildpack");
		}

		String name = entity.get("name").toString();
		String memQuota = entity.get("memory").toString();
		String state = entity.get("state").toString();
		String instances = entity.get("instances").toString();
		return new String[] { name, deduceAppTypeFromBuildpack(detectedBuildpack), memQuota, state, instances };

	}

	private Map getApplicationByAppId(String appId) throws Exception {
		String url = this.target + "/v2/apps/" + appId;
		logger.debug("url:" + url);
		WebResource webResource = restClient.resource(url);
		String response = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
				.header("Authorization", "Bearer " + this.accessToken).get(String.class);
		return new ObjectMapper().readValue(response, Map.class);
	}

	public String deduceAppTypeFromBuildpack(String detectedBuildpack) {
		String appType = Constants.APP_TYPE_UNKNOWN;

		if (detectedBuildpack != null) {
			for (int i = 0; i < appTypeMapper.length; i++) {
				if (detectedBuildpack.matches(appTypeMapper[i][1])) {
					appType = appTypeMapper[i][0];
					break;
				}
			}
		} else
			return "";

		return appType;
	}

	public int getAppInstancesByAppId(String appId) throws Exception {
		return Integer.parseInt(getAppInfoByAppId(appId)[4]);
	}

	public int getRunningInstances(String appId) throws Exception {
		Map appJsonMap = this.getApplicationSummaryByAppId(appId);
		return Integer.parseInt(appJsonMap.get("running_instances").toString());
	}

	private Map getApplicationSummaryByAppId(String appId) throws Exception {
		String url = this.target + "/v2/apps/" + appId + "/summary";
		logger.debug("url:" + url);
		WebResource webResource = restClient.resource(url);
		String response = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
				.header("Authorization", "Bearer " + this.accessToken).get(String.class);
		return new ObjectMapper().readValue(response, Map.class);
	}

	public void updateInstances(String appId, int instances) throws CloudException {
		String restUrl = this.target + "/v2/apps/{appId}?instances={instances}";
		restUrl = restUrl.replace("{appId}", appId).replace("{instances}", String.valueOf(instances));
		JSONObject jsonObj = new JSONObject();
		jsonObj.put("instances", instances);
		try {
			WebResource webResource = restClient.resource(restUrl);
			ClientResponse response = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
					.header("Authorization", "Bearer " + accessToken).put(ClientResponse.class, jsonObj.toString());
			int status = response.getStatus();
			if (String.valueOf(status).startsWith("2")) {
				return;
			}
			String content = response.getEntity(String.class);
			JSONObject json = new JSONObject(content);
			String errorCode = (String) json.get("error_code");
			String description = (String) json.get("description");
			logger.error(description);
			throw new CloudException(errorCode, description);
		} catch (Exception e) {
			throw new CloudException(e);
		}
	}

	public Map<String, Map<String, Object>> getApplicationStatsByAppId(String appId) throws IOException {
		String url = this.target + "/v2/apps/" + appId + "/stats";

		WebResource webResource = restClient.resource(url);
		String response = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
				.header("Authorization", "Bearer " + this.accessToken).get(String.class);

		JSONObject jsonObj = new JSONObject(response);
		Set<String> keySet = jsonObj.keySet();
		Map<String, Map<String, Object>> instanceStats = new HashMap<String, Map<String, Object>>(keySet.size());
		logger.debug(String.format("%d instances for app %s", keySet.size(), appId));
		for (String key : keySet) {
			String id = key;
			JSONObject jsonStats = (JSONObject) jsonObj.get(key);

			Map<String, Object> attributes = new HashMap<String, Object>();

			String state = (String) jsonStats.get("state");
			// only count in RUNNING instance
			if (!"RUNNING".equals(state)) {
				logger.warn(String.format("instace %s of %s is not RUNNING: %s ", id, appId, state));
				continue;
			}
			attributes.put("state", state);

			Map<String, Object> statsMap = new HashMap<String, Object>();
			JSONObject statsObj = (JSONObject) jsonStats.get("stats");
			statsMap.put("name", statsObj.get("name"));
			statsMap.put("host", statsObj.get("host"));
			statsMap.put("port", statsObj.get("port"));
			statsMap.put("uptime", Double.parseDouble(statsObj.get("uptime").toString()));
			statsMap.put("mem_quota", statsObj.get("mem_quota"));
			statsMap.put("disk_quota", statsObj.get("disk_quota"));
			statsMap.put("fds_quota", statsObj.get("fds_quota"));

			Map<String, Object> usageMap = new HashMap<String, Object>();
			JSONObject usageObj = (JSONObject) statsObj.get("usage");
			usageMap.put("time", usageObj.get("time"));
			usageMap.put("cpu", Double.parseDouble(usageObj.get("cpu").toString()));
			usageMap.put("mem", Double.parseDouble(usageObj.get("mem").toString()));
			usageMap.put("disk", Integer.parseInt(usageObj.get("disk").toString()));

			statsMap.put("usage", usageMap);

			attributes.put("stats", statsMap);

			instanceStats.put(id, attributes);
		}

		if (instanceStats.size() == 0) {
			instanceStats = null;
		}
		return instanceStats;
	}

}
