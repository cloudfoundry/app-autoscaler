package org.cloudfoundry.autoscaler.common.util;

import java.io.IOException;

import javax.servlet.ServletException;

import org.apache.log4j.Logger;
import org.json.JSONObject;

import com.sun.jersey.api.client.Client;
import com.sun.jersey.api.client.ClientResponse;
import com.sun.jersey.api.client.WebResource;

public class AuthenticationTool {
	private static final Logger logger = Logger.getLogger(AuthenticationTool.class);
	private Client restClient;

	private String authEndpoint;
	private String cloudControllerEndpoint;
	private boolean isSslSupported = true;
	private static AuthenticationTool instance;


	private AuthenticationTool() {
		this.restClient = RestUtil.getHTTPSRestClient();
	}

	public static AuthenticationTool getInstance() {
		if (instance == null) {
			synchronized (AuthenticationTool.class) {
				if (instance == null)
					instance = new AuthenticationTool();
			}
		}
		return instance;
	}

	public boolean isSslSupported() {
		return isSslSupported;
	}

	public void setSslSupported(boolean isSslSupported) {
		this.isSslSupported = isSslSupported;
	}

	public String getAuthEndpoint() throws IOException, ServletException {
		if (authEndpoint == null) {
			JSONObject cloudInfo = getCloudInfo(getCloudControllerEndpoint());
			authEndpoint = cloudInfo.get("authorization_endpoint").toString();
		}

		logger.debug("Get UAA Endpoint: " + authEndpoint);
		return authEndpoint;
	}

	public void setAuthEndpoint(String authEndpoint) {
		this.authEndpoint = authEndpoint;
	}

	public String getuserIdFromToken(String token) throws ServletException, IOException {
		String userInfoEndpoint = getAuthEndpoint() + "/userinfo";
		JSONObject userInfo = getCurrentUserInfo(token, userInfoEndpoint);
		logger.debug("Get user info: " + userInfo.toString());
		
		Object userId = userInfo.get("user_id");
		return userId == null? null : userId.toString();
		
	}

	private JSONObject getCurrentUserInfo(String token, String userInfoEndpoint) throws ServletException {
		String authorization = "bearer " + token;

		WebResource resource = restClient.resource(userInfoEndpoint);

		try {
			String rawUserInfo = resource.header("Authorization", authorization).get(String.class);
			JSONObject userInfo = new JSONObject(rawUserInfo);
			return userInfo;
		} catch (Exception e) {
			throw new ServletException("Get userinfo failed from endpoint " + userInfoEndpoint);
		}

	}


	public boolean isUserDeveloperOfAppSpace(String userId, String token, String appId) throws Exception {
		String ccEndPoint = getCloudControllerEndpoint();
		String url = ccEndPoint + "/v2/users/" + userId + "/spaces" + "?q=app_guid:" + appId;
		logger.debug("Get user's app space: " + url);

		WebResource resource = restClient.resource(url);
		ClientResponse cr = resource.header("Authorization", "Bearer " + token).accept("application/json")
				.get(ClientResponse.class);

		int status = cr.getStatus();
		String body = cr.getEntity(String.class);

		if ((status >= 200) && (status < 300)) {
			logger.debug("Get user's app space response:" + body);

			JSONObject spacesInfo = new JSONObject(body);
			int nResults = Integer.parseInt(spacesInfo.get("total_results").toString());
			if (nResults > 0) {
				return true;
			}
		} else {
			JSONObject json = new JSONObject(body);
			String errorCode = (String) json.get("error_code");
			String description = (String) json.get("description");
			logger.error("Get error response from  " + url + ": " + errorCode + " : " + description);
		}

		return false;
	}


	public void setCloudControllerEndpoint(String cloudControllerEndpoint) {
		this.cloudControllerEndpoint = cloudControllerEndpoint;
	}

	private String getCloudControllerEndpoint() {
		if (cloudControllerEndpoint == null) {
			String cfUrl = ConfigManager.get("cfUrl").toLowerCase();
			if (cfUrl.startsWith("http://") || cfUrl.startsWith("https://"))
				cloudControllerEndpoint = cfUrl;
			else
				cloudControllerEndpoint = "https://" + cfUrl;
		}
		logger.debug("Get Cloud Controller Endpoint: " + cloudControllerEndpoint);
		return cloudControllerEndpoint;
	}

	private JSONObject getCloudInfo(String ccEndpoint) throws ServletException, IOException {
		String infoUrl = ccEndpoint + "/v2/info";
		Client restClient = RestUtil.getHTTPSRestClient();

		WebResource infoResource = restClient.resource(infoUrl);
		logger.debug("infoUrl: " + infoUrl);
		ClientResponse infoResponse = infoResource.get(ClientResponse.class);

		if (infoResponse.getStatus() == 200) {
			return new JSONObject(infoResponse.getEntity(String.class));
		} else {
			throw new ServletException(
					"Can not get the oauth information from " + infoUrl + ", code: " + infoResponse.getStatus());
		}
	}

}
