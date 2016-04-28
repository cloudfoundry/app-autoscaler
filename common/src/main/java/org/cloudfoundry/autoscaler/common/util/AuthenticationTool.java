package org.cloudfoundry.autoscaler.common.util;


import java.io.IOException;
import java.util.Iterator;

import javax.servlet.ServletException;

import org.apache.log4j.Logger;
import org.json.JSONArray;
import org.json.JSONException;
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
    
    public enum SecurityCheckStatus {
        SECURITY_CHECK_SSO,        //internal state, request will be redirected to somewhere else. 
        SECURITY_CHECK_COMPLETE,    //final state, check is successful and user data being written into session. 
        SECURITY_CHECK_ERROR        //final state, request is redirected to error page.
    }

    private AuthenticationTool(){
   		this.restClient= RestUtil.getHTTPSRestClient();
    }

    public static AuthenticationTool getInstance(){
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
   
    public String getAuthEndpoint() throws IOException, ServletException{
        if(authEndpoint == null){
	        JSONObject cloudInfo = getCloudInfo(getCloudControllerEndpoint());
	        authEndpoint = cloudInfo.get("authorization_endpoint").toString();
        }

        logger.debug("Get UAA Endpoint: "  + authEndpoint);
        return authEndpoint;
    }

    public void setAuthEndpoint(String authEndpoint) {
        this.authEndpoint = authEndpoint;
    }
    
    public String getUserIDFromToken(String token) throws ServletException, IOException{
	        String userInfoEndpoint = getAuthEndpoint() + "/userinfo";
	        JSONObject userInfo = getCurrentUserInfo(token, userInfoEndpoint);
	        logger.debug("Get user info: "  +  userInfo.toString());
	        return userInfo.get("user_id").toString() ;     
    }
    

    private JSONObject getCurrentUserInfo(String token, String userInfoEndpoint) throws ServletException {
        String authorization = "bearer " + token;  
        
        WebResource resource = restClient.resource(userInfoEndpoint);

        try {
	        String rawUserInfo = resource.header("Authorization", authorization).get(String.class);
	        JSONObject userInfo = new JSONObject(rawUserInfo);
	        return userInfo;
        }
        catch (Exception e) {
            throw new ServletException("Get userinfo failed from endpoint " + userInfoEndpoint );
        }   
        
    }
    
    private boolean isUserOwnedSpace(String ccEndpoint, String url, String token, String spaceId) throws ServletException {
        WebResource resource;           

        resource = restClient.resource(ccEndpoint + url);
        JSONObject spacesInfo = null;
        String rawSpacesInfo = null;
        try {
            rawSpacesInfo = resource.header("Authorization", "Bearer " + token).accept("application/json").get(String.class);
            spacesInfo = new JSONObject(rawSpacesInfo);
        }
        catch (JSONException e) {
            throw new ServletException("Parsing response from space info failed: " + rawSpacesInfo);
        }

        JSONArray spaces = (JSONArray) spacesInfo.get("resources");
        if ((spaces != null) && (spaces.length() > 0)) {
            Iterator<?> iterSpaces = spaces.iterator();
            while (iterSpaces.hasNext()) {
                JSONObject space = (JSONObject) iterSpaces.next();
                JSONObject spaceMeta = (JSONObject) space.get("metadata");
                String spaceGuid = spaceMeta.get("guid").toString();

                if (spaceGuid.equals(spaceId))
                    return true;
            }
        }
        
        Object nextUrl = spacesInfo.get("next_url");
        if ( nextUrl != null && ! nextUrl.toString().equals("null") && !nextUrl.toString().isEmpty()){
            return isUserOwnedSpace(ccEndpoint, nextUrl.toString(), token, spaceId);
        }

        return false;
    }
    
    
    private boolean isSpaceUser(String ccEndpoint, String token, String userID, String spaceId) throws ServletException {
        String userUrl = "/v2/users/" + userID;
        
        String spacesUrl =  userUrl + "/spaces";
        String managedSpacesUrl = userUrl + "/managed_spaces";
        String auditedSpacesUrl = userUrl + "/audited_spaces";
        
        logger.debug("userUrl: " + userUrl);

        if (isUserOwnedSpace(ccEndpoint, spacesUrl, token, spaceId)
            || isUserOwnedSpace(ccEndpoint, managedSpacesUrl, token, spaceId)
            || isUserOwnedSpace(ccEndpoint, auditedSpacesUrl, token, spaceId)){
            return true;
        } else {
            return false;
        }
    }

 
    public SecurityCheckStatus doValidateToken(String userID, String token, String org_id, String space_id)throws ServletException, IOException{

    	String ccEndpoint = getCloudControllerEndpoint();        
                 
        if (isSpaceUser(ccEndpoint, token, userID, space_id)) {
        	return SecurityCheckStatus.SECURITY_CHECK_COMPLETE;
        } 
        else{
            throw new ServletException("Current user has no permission in current space.");
        }    

    	
    }



    public void setCloudControllerEndpoint(String cloudControllerEndpoint) {
        this.cloudControllerEndpoint = cloudControllerEndpoint;
    }
    
    private String getCloudControllerEndpoint() {              
        if(cloudControllerEndpoint == null) {
	        String cfUrl = ConfigManager.get("cfUrl").toLowerCase();
	        if (cfUrl.startsWith("http://") || cfUrl.startsWith("https://"))
		        cloudControllerEndpoint = cfUrl;
	        else    
	        	cloudControllerEndpoint = "https://" + cfUrl ;
        }
        logger.debug("Get Cloud Controller Endpoint: "  + cloudControllerEndpoint);
        return cloudControllerEndpoint;
    }

    private JSONObject getCloudInfo(String ccEndpoint) throws ServletException, IOException{
        String infoUrl = ccEndpoint + "/v2/info";
        Client restClient = RestUtil.getHTTPSRestClient();  

        WebResource infoResource = restClient.resource(infoUrl);
        logger.debug("infoUrl: " + infoUrl);
        ClientResponse infoResponse = infoResource.get(ClientResponse.class);

        if(infoResponse.getStatus() == 200){
            return new JSONObject(infoResponse.getEntity(String.class));
        } else {
            throw new ServletException("Can not get the oauth information from " + infoUrl + ", code: " + infoResponse.getStatus());
        }
    }
 
}
