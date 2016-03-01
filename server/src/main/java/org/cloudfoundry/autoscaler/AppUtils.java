/**
 * 
 */
package org.cloudfoundry.autoscaler;

import java.util.HashMap;
import java.util.Map;

import javax.ws.rs.core.MediaType;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.exceptions.AppNotFoundException;
import org.cloudfoundry.autoscaler.exceptions.AuthorizationException;
import org.cloudfoundry.autoscaler.exceptions.OrgNotFoundException;
import org.cloudfoundry.autoscaler.exceptions.SpaceNotFoundException;
import org.json.JSONArray;
import org.json.JSONObject;

import com.sun.jersey.api.client.Client;
import com.sun.jersey.api.client.WebResource;

/**
 * 
 * Utility methods
 * 
 * @author paolo
 * 
 */
public class AppUtils {

	private static final String CLASS_NAME = AppUtils.class.getName();
	private static final Logger logger = Logger.getLogger(CLASS_NAME);
	public static final String ORG_NAME="ORG_NAME";
	public static final String ORG_GUID="ORG_GUID";	
	public static final String SPACE_NAME="SPACE_NAME";
	public static final String SPACE_GUID="SPACE_GUID";
	public static final String APP_GUID="APP_GUID";
	

  /**
   * Given an app name and version, finds the org and space to which app belongs using directly the CF REST API, assuming
   * that the account used to interact with the REST API has been granted visibility on the target org and space.
   * 
   * @param cfApiTarget
   * @param cfUser
   * @param cfPassword
   * @param appName
   * @param appVersion
   * @return
   * @throws AuthorizationException
   * @throws AppNotFoundException
   * @throws OrgNotFoundException
   * @throws SpaceNotFoundException
   */
	public static Map<String, String> findOrgSpaceForApp(String cfApiTarget, String cfUser, String cfPassword,
			String appName, String appVersion) throws AuthorizationException,AppNotFoundException,OrgNotFoundException,SpaceNotFoundException {
		Map<String, String> result=null;
		String access_token;
		String appGuid;
		String spaceName;
		String spaceGuid;
		// get OAuth2 Token
		try {
			String url = "http://" + cfApiTarget + "/info";
			logger.debug( "connecting to URL:" + url);
			
			WebResource webResource = Client.create().resource(url);
			String response = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON).get(String.class);

			logger.debug( ">>>" + response);
			JSONObject jobj = new JSONObject(response);
			String authorization_endpoint = (String) jobj.get("authorization_endpoint");
			logger.debug( ">>>" + authorization_endpoint);
			webResource = Client.create().resource(authorization_endpoint + "/oauth/token");
			response = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_FORM_URLENCODED).header("charset", "utf-8")
			.header("authorization", "Basic Y2Y6").post(String.class, "grant_type=password&username=" + cfUser + "&password="
					+ cfPassword);

			logger.debug( ">>>" + response);
			jobj = new JSONObject(response);
			access_token = (String) jobj.get("access_token");
			logger.debug( ">>>" + access_token);
		} catch (Exception  e) {
			throw new AuthorizationException("userId=" + cfUser + " " + e.getMessage());
		}
			
		// Query to find matching apps (with same name, but may belong to other spaces/orgs, then match on version)
		try{
			String url = "http://" + cfApiTarget + "/v2/apps?q=name:" + appName;
			logger.debug( "connecting to URL:" + url);
			WebResource webResource = Client.create().resource(url);
			String response = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON).
					header("Authorization", "Bearer " + access_token).get(String.class);
//			res = client.resource(url).accept(MediaType.APPLICATION_JSON).contentType(MediaType.APPLICATION_JSON)
//					.header("Authorization", "Bearer " + access_token);
//			String response = res.get(String.class);
			logger.debug( ">>>" + response);

			JSONObject jobj = new JSONObject(response);
			JSONArray jarray = (JSONArray) jobj.get("resources");
			appGuid = null;
			for (Object obj : jarray) {
				String version = (String) ((JSONObject) ((JSONObject) obj).get("entity")).get("version");
				logger.debug( ">>>> " + version);
				if (appVersion.equals(version)) {
					appGuid = (String) ((JSONObject) ((JSONObject) obj).get("metadata")).get("guid");
					break;
				}
			}
			if (appGuid == null) {
				throw new Exception("Could not find GUID for app: "+appName);
			}
			
			
			String restUrl = "http://" + cfApiTarget  + "/v2/apps/{appId}?inline-relations-depth=1";
			restUrl = restUrl.replace("{appId}", appGuid);
			webResource = Client.create().resource(restUrl);
			response = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON).
					header("Authorization", "Bearer " + access_token).get(String.class);
			JSONObject ooo = new JSONObject(response);
			
			restUrl = "http://" + cfApiTarget  + "/v2/apps/{appId}/stats";
			restUrl = restUrl.replace("{appId}", appGuid);
			webResource = Client.create().resource(restUrl);
			response = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON).
					header("Authorization", "Bearer " + access_token).get(String.class);
			ooo = new JSONObject(response);
			
		} catch (Exception e) {
			throw new AppNotFoundException("appId=" + appName + " " + e.getMessage());
		}
			
		// find matching space for App GUID
		try{				
			String url = "http://" + cfApiTarget + "/v2/spaces?q=app_guid:" + appGuid;
			logger.debug( "connecting to URL:" + url);
			WebResource webResource = Client.create().resource(url);
			String response = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON).
					header("Authorization", "Bearer " + access_token).get(String.class);

			logger.debug( ">>>" + response);
			JSONObject jobj = new JSONObject(response);
			JSONArray jarray = (JSONArray) jobj.get("resources");
			if (jarray.length() != 1) {
				throw new Exception("Could not find matching space for app: "+appName);
			}
			spaceGuid = (String) ((JSONObject) ((JSONObject) jarray.get(0)).get("metadata")).get("guid");
			spaceName = (String) ((JSONObject) ((JSONObject) jarray.get(0)).get("entity")).get("name");
			if (spaceGuid == null || spaceName == null) {
				throw new Exception("Could not find  space for app: "+appName);
			}
		} catch (Exception e) {
			throw new SpaceNotFoundException("appId=" + appName + " " + e.getMessage());
		}

		// finally find the org name to which the space belongs
		try{
			String url = "http://" + cfApiTarget + "/v2/organizations?q=space_guid:" + spaceGuid;
			logger.debug( "connecting to URL:" + url);
			WebResource webResource = Client.create().resource(url);
			String response = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON).
					header("Authorization", "Bearer " + access_token).get(String.class);
			logger.debug( ">>>" + response);
			JSONObject jobj = new JSONObject(response);
			JSONArray jarray = (JSONArray) jobj.get("resources");
			if (jarray.length() != 1) {
				throw new Exception("Could not find matching organization for app: "+appName);
			}
			String orgGuid = (String) ((JSONObject) ((JSONObject) jarray.get(0)).get("metadata")).get("guid");
			String orgName = (String) ((JSONObject) ((JSONObject) jarray.get(0)).get("entity")).get("name");
			if (orgName == null || orgGuid == null) {
				throw new Exception("Could not find organization for app: "+appName);
			}
			logger.info( ">>> Found org_name =" + orgName + " space_name=" + spaceName + " for app " + appName);
			result=new HashMap<String, String>();
			result.put(SPACE_GUID, spaceGuid);
			result.put(SPACE_NAME, spaceName);
			result.put(ORG_GUID, orgGuid);
			result.put(ORG_NAME, orgName);
			result.put(APP_GUID, appGuid);			
		} catch (Exception e) {
			throw new OrgNotFoundException("appId=" + appName + " " + e.getMessage());
		}

		return result;
	}
	

}
