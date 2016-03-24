package org.cloudfoundry.autoscaler.util;

import java.io.IOException;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Set;

import javax.ws.rs.core.MediaType;

import org.apache.commons.codec.binary.Base64;
import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.constant.Constants;
import org.cloudfoundry.autoscaler.exceptions.CloudException;
import org.cloudfoundry.autoscaler.metric.bean.CFInstanceStats;
import org.cloudfoundry.autoscaler.metric.bean.CloudAppInstance;
import org.cloudfoundry.autoscaler.metric.bean.InstanceState;
import org.cloudfoundry.autoscaler.metric.bean.CFInstanceStats.Usage;
import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.sun.jersey.api.client.Client;
import com.sun.jersey.api.client.ClientResponse;
import com.sun.jersey.api.client.WebResource;

@SuppressWarnings({ "rawtypes", "unchecked" })
public class CloudFoundryManager {
    private static final Logger logger = Logger.getLogger(CloudFoundryManager.class);
    public static final String ORG_NAME = "ORG_NAME";
    public static final String ORG_GUID = "ORG_GUID";
    public static final String SPACE_NAME = "SPACE_NAME";
    public static final String SPACE_GUID = "SPACE_GUID";
    public static final String APP_GUID = "APP_GUID";
    public static final String USERNAME = "userName";
    public static final String PASSWORD = "password";
    public static final String HOST = "host";
    public static final String PORT = "port";
    public static final String HEADERUAAENDPOINT = "uaaEndpoint";
    public static final String HEADERACCESSTOKEN = "x-accessToken";
    public static final String HEADERACCESSTOKENEXPIREINTERVAL = "x-accessTokenExpireInterval";
    public static final String HEADERTOKENSESSIONID = "TOKENSESSIONID";
    
    private String target;
    private String accessToken;
    private long accessTokenExpireTime;
    private long accessTokenGenerateTime;
    private long accessTokenExpireInterval;
    private String accessTokenFilterSessionId;
    private String cfClientId;
    private String cfSecretKey;
    private Client restClient;

    
    private static  String[][] appTypeMapper = { 
		{Constants.APP_TYPE_JAVA, "(?i).*Liberty.*"},
		{Constants.APP_TYPE_RUBY_ON_RAILS, "(?i).*Ruby/Rails.*"},
		{Constants.APP_TYPE_RUBY_SINATRA,"(?i).*Ruby/Rack.*"},
		{Constants.APP_TYPE_RUBY, "(?i).*Ruby.*"},
		{Constants.APP_TYPE_NODEJS, "(?i).*(Node\\.js|nodejs).*"},
		{Constants.APP_TYPE_GO, "(?i).*go.*"},
		{Constants.APP_TYPE_PHP, "(?i).*php.*"},
		{Constants.APP_TYPE_PYTHON, "(?i).*python.*"},
		{Constants.APP_TYPE_DOTNET, "(?i).*dotnet.*"},
    };
    
    private static volatile CloudFoundryManager instance;

    private CloudFoundryManager() {
        this.cfClientId = getClientId();
        this.cfSecretKey = getSecretKey();
        this.target = "https://" + getCFAPIUrl();
   		this.restClient= RestUtil.getHTTPSRestClient();
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
        
            String authString = cfClientId + ":" + cfSecretKey;
            byte[] authEncBytes = Base64.encodeBase64(authString.getBytes());
            String authStringEnc = new String(authEncBytes);
            Map jobj = new ObjectMapper().readValue(response, Map.class);
            // String authorization_endpoint = (String) jobj.get("token_endpoint");
            String authorization_endpoint = (String) jobj.get("authorization_endpoint");           
            webResource = restClient.resource(authorization_endpoint + "/oauth/token");
            ClientResponse cr = webResource
                    .accept(MediaType.APPLICATION_JSON)
                    .type(MediaType.APPLICATION_FORM_URLENCODED)
                    .header("charset", "utf-8")
                    .header("authorization", "Basic " + authStringEnc)
                    .post(ClientResponse.class,
                            "grant_type=client_credentials&client_id=" + cfClientId + "&client_secret=" + cfSecretKey);
            response = cr.getEntity(String.class);
            
            jobj = new ObjectMapper().readValue(response, Map.class);

            accessToken = (String) jobj.get("access_token");
            long expire_in = Long.parseLong(jobj.get("expires_in").toString());
            accessTokenExpireTime = System.currentTimeMillis() + expire_in * 1000;
            this.accessTokenExpireInterval = expire_in * 1000;
            this.accessTokenExpireTime = System.currentTimeMillis() + expire_in * 1000;
            this.accessTokenGenerateTime = System.currentTimeMillis();
            this.accessTokenFilterSessionId = null;


    }
    public void setAccessFilterSessionId(String sessionId){
    	
    	this.accessTokenFilterSessionId = sessionId;
    }
    public String getAccessFilterSessionId(){
    	return this.accessTokenFilterSessionId;
    }
    public long getAccessTokenExpireInterval(){
    	return this.accessTokenExpireInterval - (System.currentTimeMillis() - this.accessTokenGenerateTime);
    }
    public String getAccessToken(){
    	return this.accessToken;
    }
    public void refreshAccessToken(){
    	try {
			this.loginWithClientId();
		} catch (Exception e) {
			// TODO Auto-generated catch block
			e.printStackTrace();
		}
    }
    public JSONObject getRunningEnvVariableGroup()throws Exception{
    	 String url = this.target + "/v2/config/environment_variable_groups/running";
         logger.debug("url:" + url);
         WebResource webResource = restClient.resource(url);
         String response = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
                 .header("Authorization", "Bearer " + this.accessToken).get(String.class);
         return  new JSONObject(response);
    	
    }
    // get application statistics
    public List<CloudAppInstance> getAppStatsExtByAppId(String appId) throws Exception {

        logger.debug("Calling CF to get stats of app " + appId);
        List<CFInstanceStats> statsList = getApplicationStatsByAppId(appId);
        if (statsList == null) {
            return null;
        }
        ArrayList<CloudAppInstance> resultList = new ArrayList<CloudAppInstance>();
        for (CFInstanceStats instStats : statsList) {
            double cpuPerc = 0;
            double memMB = 0;
            Usage instUsage = instStats.getUsage();
            double memQuotaMB = instStats.getMemQuota() / (1024.0 * 1024.0);
            long timestamp = System.currentTimeMillis();
            if (instUsage != null) {
                cpuPerc = 100 * instUsage.getCpu();
                memMB = instUsage.getMem() / (1024.0 * 1024.0);
                timestamp = instUsage.getTime().getTime();
            }
            CloudAppInstance resultStats = new CloudAppInstance(instStats.getId(), instStats.getHost(),
                    instStats.getCores(), cpuPerc, memMB, memQuotaMB, timestamp);
            logger.debug(String.format("inst = %16s  cores = %2d  cpu = %6.1f %%  mem = %6.1f MB mem_quota = %6.1f MB",
                    instStats.getId(), instStats.getCores(), cpuPerc, memMB, memQuotaMB));
            resultList.add(resultStats);
        }
        return resultList;

    }

    private List<CFInstanceStats> getApplicationStatsByAppId(String appId) throws IOException {
        List<CFInstanceStats> statsList = new ArrayList<CFInstanceStats>();
        String url = this.target + "/v2/apps/" + appId + "/stats";

        WebResource webResource = restClient.resource(url);
        String response = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
                .header("Authorization", "Bearer " + this.accessToken).get(String.class);

        JSONObject jsonObj = new JSONObject(response);
        Set<String> keySet = jsonObj.keySet();
        logger.debug(String.format("%d instances for app %s", keySet.size(), appId));
        for (String key : keySet) {
            Object id = key;
            JSONObject jsonStats = (JSONObject) jsonObj.get(key);

            Map<String, Object> attributes = new HashMap<String, Object>();

            String state = (String) jsonStats.get("state");
            // only count in RUNNING instance
            if (!InstanceState.RUNNING.equals(InstanceState.valueOf(state))) {
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

            CFInstanceStats stats = new CFInstanceStats(id.toString(), attributes);
            statsList.add(stats);
        }

        if (statsList.size() == 0) {
            statsList = null;
        }
        return statsList;
    }

    
    public int getRunningInstances(String appId) throws Exception{

        Map appJsonMap = this.getApplicationRunnningStanceByAppId(appId);
        return  Integer.parseInt(appJsonMap.get("running_instances").toString());
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
            String response = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
                    .header("Authorization", "Bearer " + this.accessToken).get(String.class);
           
            logger.debug(">>>" + response);
            JSONObject jobj = new JSONObject(response);
            JSONArray jarray = (JSONArray) jobj.get("resources");
            if (jarray.length() != 1) {
                throw new Exception("Could not find matching space for app: " + appId);
            }
            spaceGuid = (String) ((JSONObject) ((JSONObject) jarray.get(0)).get("metadata")).get("guid");
            spaceName = (String) ((JSONObject) ((JSONObject) jarray.get(0)).get("entity")).get("name");
            if (spaceGuid == null || spaceName == null) {
                throw new Exception("Could not find  space for app: " + appId);
            }
        } catch (Exception e) {
            throw new Exception("appId=" + appId + " " + e.getMessage());
        }

        // finally find the org name to which the space belongs
        try {
            String url = this.target + "/v2/organizations?q=space_guid:" + spaceGuid;
            logger.debug("connecting to URL:" + url);
            WebResource webResource = restClient.resource(url);
            String response = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
                    .header("Authorization", "Bearer " + this.accessToken).get(String.class);
            
            logger.info(">>>" + response);
            JSONObject jobj = new JSONObject(response);
            JSONArray jarray = (JSONArray) jobj.get("resources");
            if (jarray.length() != 1) {
                throw new Exception("Could not find matching organization for app: " + appId);
            }
            String orgGuid = (String) ((JSONObject) ((JSONObject) jarray.get(0)).get("metadata")).get("guid");
            String orgName = (String) ((JSONObject) ((JSONObject) jarray.get(0)).get("entity")).get("name");
            if (orgName == null || orgGuid == null) {
                throw new Exception("Could not find organization for app: " + appId);
            }
            logger.info(">>> Found org_name =" + orgName + " space_name=" + spaceName + " for app " + appId);
            result = new HashMap<String, String>();
            result.put(SPACE_GUID, spaceGuid);
            result.put(SPACE_NAME, spaceName);
            result.put(ORG_GUID, orgGuid);
            result.put(ORG_NAME, orgName);
            result.put(APP_GUID, appId);
        } catch (Exception e) {
            throw new Exception("appId=" + appId + " " + e.getMessage());
        }

        return result;
    }
    
    
    public String getAppNameByAppId(String appId) throws Exception {
        return getAppInfoByAppId(appId)[0];
    }
    
    public String[] getAppNameAndType(String appId) throws Exception {
        String [] appInfo = getAppInfoByAppId(appId);
        return new String[] {appInfo[0], appInfo[1]};
    }
    
    public String getAppType(String appId) throws Exception {
        return getAppInfoByAppId(appId)[1];
    }

    public String getAppMemQuotaByAppId(String appId) throws Exception {
        return getAppInfoByAppId(appId)[2];
    }

    public String getAppStateByAppId(String appId) throws Exception {
        return getAppInfoByAppId(appId)[3];
    }
    
    public int getAppInstancesByAppId(String appId) throws Exception {
        return Integer.parseInt(getAppInfoByAppId(appId)[4]);
    }
    
    
    public String[] getAppInfoByAppId(String appId) throws Exception  {
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
        return new String[] { name, deduceAppTypeFromBuildpack(detectedBuildpack), memQuota, state, instances};
        
    }
    
    private Map getApplicationByAppId(String appId) throws Exception {
        String url = this.target + "/v2/apps/" + appId;
        logger.debug("url:" + url);
        WebResource webResource = restClient.resource(url);
        String response = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
                .header("Authorization", "Bearer " + this.accessToken).get(String.class);
        return new ObjectMapper().readValue(response, Map.class);
    }
    
    private Map getApplicationRunnningStanceByAppId(String appId) throws Exception {
        String url = this.target + "/v2/apps/" + appId + "/summary";
        logger.debug("url:" + url);
        WebResource webResource = restClient.resource(url);
        String response = webResource.accept(MediaType.APPLICATION_JSON).type(MediaType.APPLICATION_JSON)
                .header("Authorization", "Bearer " + this.accessToken).get(String.class);
        return new ObjectMapper().readValue(response, Map.class);
    }
    
    public static String getCFAPIUrl() {

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

    public String deduceAppTypeFromBuildpack(String detectedBuildpack) {
        String appType = Constants.APP_TYPE_UNKNOWN;
        
        if (detectedBuildpack != null){
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

    private String getIdFromJson(String json, String name) throws Exception {
        Map jsonMap = new ObjectMapper().readValue(json, Map.class);
        String id = null;
        List records = (List) jsonMap.get("resources");
        for (Object record : records) {
            Map metadata = (Map) ((Map) record).get("metadata");
            Map entity = (Map) ((Map) record).get("entity");
            if (name.equals(entity.get("name"))) {
                id = metadata.get("guid").toString();
                break;
            }
        }
        return id;
    }

    
	public void updateInstances(String appId, int instances)  throws CloudException {
		String restUrl = this.target + "/v2/apps/{appId}?instances={instances}";
		restUrl = restUrl.replace("{appId}", appId).replace("{instances}",
				String.valueOf(instances));
		JSONObject jsonObj = new JSONObject();
		jsonObj.put("instances", instances);
		try {
			WebResource webResource = restClient.resource(restUrl);
			ClientResponse response = webResource
				.accept(MediaType.APPLICATION_JSON)
				.type(MediaType.APPLICATION_JSON)
				.header("Authorization", "Bearer " + accessToken)
				.put(ClientResponse.class, jsonObj.toString());
			int status = response.getStatus();
			if (String.valueOf(status).startsWith("2")) {
				return;
			}
			String content = response.getEntity(String.class);
			JSONObject json = new JSONObject(content);
			String errorCode = (String)json.get("error_code");
			String description = (String)json.get("description");
			logger.error(description);
			throw new CloudException(errorCode, description);
		} catch (Exception e) {
			throw new CloudException(e);
		}
	}
    
    
    
}
