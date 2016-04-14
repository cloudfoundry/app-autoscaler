package org.cloudfoundry.autoscaler.api.util;

import java.io.IOException;
import java.util.HashMap;
import java.util.Map;

import javax.ws.rs.core.MediaType;

import org.apache.commons.codec.binary.Base64;
import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.api.Constants;
import org.cloudfoundry.autoscaler.api.exceptions.AppInfoNotFoundException;
import org.cloudfoundry.autoscaler.api.exceptions.AppNotFoundException;
import org.cloudfoundry.autoscaler.api.exceptions.CloudException;
import org.cloudfoundry.autoscaler.api.exceptions.ServiceNotFoundException;
import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.JsonNode;

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

    
    private static volatile CloudFoundryManager instance;

    private CloudFoundryManager() {
        this.cfClientId = getClientId();
        this.cfSecretKey = getSecretKey();
   		this.restClient= RestUtil.getHTTPSRestClient();
   		String cfUrlTemp = getCFAPIUrl();
   		if(!cfUrlTemp.startsWith("http://") || cfUrlTemp.startsWith("https://")){
   			this.target = "https://" + getCFAPIUrl();
   		}
   		else{
   			this.target = getCFAPIUrl();
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
    
   

    private void login() throws Exception {
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
            byte[] authEncBytes = Base64.encodeBase64(authString.getBytes());
            String authStringEnc = new String(authEncBytes);
            JsonNode jobj = new ObjectMapper().readTree(response);
            String authorization_endpoint = jobj.get("authorization_endpoint").asText();

            logger.debug(">>>" + authorization_endpoint);
            webResource = restClient.resource(authorization_endpoint + "/oauth/token");
            ClientResponse cr = webResource
                    .accept(MediaType.APPLICATION_JSON)
                    .type(MediaType.APPLICATION_FORM_URLENCODED)
                    .header("charset", "utf-8")
                    .header("authorization", "Basic " + authStringEnc)
                    .post(ClientResponse.class,
                            "grant_type=client_credentials&client_id=" + cfClientId + "&client_secret=" + cfSecretKey);
            response = cr.getEntity(String.class);
            logger.debug(">>>" + response);
            jobj = new ObjectMapper().readTree(response);

            accessToken = jobj.get("access_token").asText();
            long expire_in = Long.parseLong(jobj.get("expires_in").toString());
            accessTokenExpireTime = System.currentTimeMillis() + expire_in * 1000;

            logger.debug(">>>" + accessToken);
  

    }
    
    public JsonNode getServiceInfo(String appId, String serviceName) throws Exception {
    	 try{
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

    	  }  
    	  catch (CloudException e) {
  			throw new CloudException(e);
  		  }
    	  catch (AppNotFoundException e){
    		  throw new AppNotFoundException(e.getAppId(), e.getMessage());  
    	  }
    	  catch (IndexOutOfBoundsException e){
    		  throw new ServiceNotFoundException(serviceName, appId);
    	  }
    	  catch (NullPointerException e){
   		      throw new ServiceNotFoundException(serviceName, appId);
   	      }
    	  catch (Exception e){ 
    		  logger.info(e.getClass().getSimpleName() + " happend");
    		  throw new Exception("error: failed to get service information from VCAP_SERVICE for appId " + appId );
    	}
    }

    public Map<String, String> findOrgSpaceForApp(String appName, String appVersion) throws Exception {

        String appGuid = null;
        
        // Query to find matching apps (with same name, but may belong to other spaces/orgs, then match on version)
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
	        if (cr.getStatus() == 404) { //404 will never be returned
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
        }
        catch (AppNotFoundException e) {
  			throw new AppNotFoundException(e.getAppId(), e.getMessage());
	    }
	    catch (AppInfoNotFoundException e) {
  			throw new AppInfoNotFoundException(e.getAppId(), e.getMessage());
	    }
        catch (Exception e) {
            throw new Exception("appId=" + appId + " " + e.getMessage());
        }

        // finally find the org name to which the space belongs
        try {
            String url = this.target + "/v2/organizations?q=space_guid:" + spaceGuid;
            logger.debug("connecting to URL:" + url);
            WebResource webResource = restClient.resource(url);
            ClientResponse cr = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
                      .header("Authorization", "Bearer " + this.accessToken).get(ClientResponse.class);
            if (cr.getStatus() == 404) { //404 will never be returned
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
                //throw new Exception("Could not find organization for app: " + appId);
                throw new AppInfoNotFoundException(appId, "Could not find organization for app");
            }
            logger.info(">>> Found org_name =" + orgName + " space_name=" + spaceName + " for app " + appId);
            result = new HashMap<String, String>();
            result.put(SPACE_GUID, spaceGuid);
            result.put(SPACE_NAME, spaceName);
            result.put(ORG_GUID, orgGuid);
            result.put(ORG_NAME, orgName);
            result.put(APP_GUID, appId);
        }
        catch (AppNotFoundException e) {
  			throw new AppNotFoundException(e.getAppId(), e.getMessage());
	    }
	    catch (AppInfoNotFoundException e) {
  			throw new AppInfoNotFoundException(e.getAppId(), e.getMessage());
	    }
        catch (Exception e) {
            throw new Exception("appId=" + appId + " " + e.getMessage());
        }

        return result;
    }
    
    public boolean check_token(String token) throws Exception {
    	String authorization_endpoint = this.getUAAendpoint();
    	logger.info(">>>" + authorization_endpoint);
    	WebResource webResource = restClient.resource(authorization_endpoint + "/check_token");
        String authString = cfClientId + ":" + cfSecretKey;
        byte[] authEncBytes = Base64.encodeBase64(authString.getBytes());
        Form form = new Form();
        form.add("token", token);
        String authStringEnc = new String(authEncBytes);
        ClientResponse cr = webResource
                .accept(MediaType.APPLICATION_JSON)
                .type(MediaType.APPLICATION_FORM_URLENCODED)
                .header("charset", "utf-8")
                .header("authorization", "Basic " + authStringEnc)
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

  	   String key=Constants.CFURL;
  	   String cfUrl = System.getenv(key);
  	  
  	   if ((cfUrl== null) || cfUrl.isEmpty() ) {
  			try {
  				String ApplicationEnvString = System.getenv("VCAP_APPLICATION");
  				if (ApplicationEnvString != null) {
  					JSONObject applicationEnv = new JSONObject(ApplicationEnvString);
  					JSONArray applicationUris = (JSONArray) applicationEnv
  							.get("application_uris");
  					if (applicationUris.length() > 0) {
  						String applicationUri = (String) applicationUris.get(0);
  						cfUrl = "api." + applicationUri.substring(applicationUri
  								.indexOf(".") + 1).trim();
  					}
  				} 			
  			} catch (JSONException e) {
  				logger.error(e.getMessage(), e);
  			}
         } 

  	   if ((cfUrl== null) || cfUrl.isEmpty() ) {
  		   cfUrl = ConfigManager.get(key);
  	   }

  	   return cfUrl;

    }

    private static String getClientId() {
        return ConfigManager.get("cfClientId");
    }

    private static String getSecretKey() {
        return ConfigManager.get("cfClientSecret");
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
        JSONArray records = (JSONArray)jsonMap.get("resources");
        for (int index = 0; index < records.length(); index++) {
            id = (String) ((JSONObject) ((JSONObject) records.get(index)).get("metadata")).get("guid");
            String Name = (String) ((JSONObject) ((JSONObject) records.get(index)).get("entity")).get("name");
            if (name.equals(Name)) {
                break;
            }
        }
        return id;
    }

    
}

