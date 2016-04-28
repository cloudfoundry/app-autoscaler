package org.cloudfoundry.autoscaler.api.filter;

import java.io.IOException;
import java.util.ArrayList;
import java.util.Map;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

import javax.servlet.Filter;
import javax.servlet.FilterChain;
import javax.servlet.FilterConfig;
import javax.servlet.ServletException;
import javax.servlet.ServletRequest;
import javax.servlet.ServletResponse;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.api.util.MessageUtil;
import org.cloudfoundry.autoscaler.api.util.RestApiResponseHandler;
import org.cloudfoundry.autoscaler.common.AppInfoNotFoundException;
import org.cloudfoundry.autoscaler.common.AppNotFoundException;
import org.cloudfoundry.autoscaler.common.ClientIDLoginFailedException;
import org.cloudfoundry.autoscaler.common.util.AuthenticationTool;
import org.cloudfoundry.autoscaler.common.util.CloudFoundryManager;
import org.cloudfoundry.autoscaler.common.util.LocaleUtil;
import org.cloudfoundry.autoscaler.common.util.AuthenticationTool.SecurityCheckStatus;
/**
 * Servlet Filter implementation class SSOFilter
 */

public class AuthenticationFilter implements Filter {
    private static final Logger logger = Logger.getLogger(AuthenticationFilter.class.getName());
    private static final Pattern pattern = Pattern.compile("/apps/[^/]+/");
    private final ArrayList<String> excludePatterns = new ArrayList<String>();

 
    @Override
	public void destroy() {
		
	}

    @Override
	public void doFilter(ServletRequest request, ServletResponse response, FilterChain filter) throws IOException, ServletException {

        HttpServletResponse httpResp = (HttpServletResponse) response;
        HttpServletRequest httpReq = (HttpServletRequest) request;

        String uri = httpReq.getRequestURI().toString();
        if (excludePatterns.contains(uri)) {
            filter.doFilter(httpReq, httpResp);

        } else {
        	
            String authorization = httpReq.getHeader("Authorization");
            if ( authorization == null){
            	logger.info("No Authorization header found.");
                httpResp.sendError(HttpServletResponse.SC_UNAUTHORIZED);
                return;            	
            }
            
        	if (authorization.startsWith("Bearer "))
        		authorization = authorization.replaceFirst("Bearer ", "");
        	else if (authorization.startsWith("bearer "))
        		authorization = authorization.replaceFirst("bearer ", "");
        	
        	String userID = null ;
        	
        	try {
        		userID  = AuthenticationTool.getInstance().getUserIDFromToken(authorization);
        		if ( userID == null ){
        			httpResp.sendError(HttpServletResponse.SC_UNAUTHORIZED);
        			return ;
        		}
        	}
        	catch(ServletException e){
    			logger.error("Get exception: " + e.getClass().getName() + " with message " + e.getMessage());
				httpResp.sendError(HttpServletResponse.SC_UNAUTHORIZED);
				return ;
        	}
        	catch(Exception e){
        		logger.error("Get exception: " + e.getClass().getName() + " with message " + e.getMessage());
                httpResp.sendError(HttpServletResponse.SC_INTERNAL_SERVER_ERROR);
                return;   		
        	}

        	
            String appGuid = null;
        	String orgGuid=null;
            String spaceGuid=null;
            
    		Matcher matcher = pattern.matcher(uri);
    		if (!matcher.find()){
				httpResp.sendError(HttpServletResponse.SC_BAD_REQUEST);
				return;    			
    		}	
    		
    		appGuid = matcher.group(0).replace("/apps/", "").replace("/", "");
    		try {
    			Map <String, String> orgSpace = CloudFoundryManager.getInstance().getOrgSpaceByAppId(appGuid);
        	    orgGuid = orgSpace.get("ORG_GUID");
        	    spaceGuid = orgSpace.get("SPACE_GUID");
    		}
    		catch (AppNotFoundException e){
    			httpResp.sendError(HttpServletResponse.SC_NOT_FOUND, RestApiResponseHandler.getErrorMessage(e, LocaleUtil.getLocale(httpReq)));
    			return;
    		}
    		catch (AppInfoNotFoundException e){
    			httpResp.sendError(HttpServletResponse.SC_BAD_REQUEST, RestApiResponseHandler.getErrorMessage(new AppNotFoundException(e.getAppId()), LocaleUtil.getLocale(httpReq)));
    			return;
    		}
    		catch ( ClientIDLoginFailedException e){
    			logger.error("login UAA with client ID " + e.getClientID() + " failed");
    			httpResp.sendError(HttpServletResponse.SC_INTERNAL_SERVER_ERROR, MessageUtil.getMessageString(MessageUtil.RestResponseErrorMsg_internal_server_error, LocaleUtil.getLocale(httpReq)));
    			return;    				   				
    		}
    		catch (Exception e){
        		logger.error("Get exception: " + e.getClass().getName() + " with message " + e.getMessage());
    			httpResp.sendError(HttpServletResponse.SC_INTERNAL_SERVER_ERROR);
                return;
        	}

    		logger.debug("Authentication check for appGuid " + appGuid + " with access token: " + authorization);
    		
    		try {
                
    			SecurityCheckStatus securityCheckStatus =  AuthenticationTool.getInstance().doValidateToken(userID, authorization, orgGuid, spaceGuid);
               	
    			logger.debug("Authentication status result: " + securityCheckStatus.toString());
                
    			if (securityCheckStatus == SecurityCheckStatus.SECURITY_CHECK_COMPLETE) {
    				logger.debug("SecurityCheck succeeded.");
    				filter.doFilter(request, response);
    			}
    			else{
    				logger.debug("SecurityCheck failed.");
        			httpResp.sendError(HttpServletResponse.SC_UNAUTHORIZED);
    	            return;    				
    			}
    		}catch (ServletException e) { 
    			logger.error(e.getMessage(), e);
    			httpResp.sendError(HttpServletResponse.SC_UNAUTHORIZED);
	            return;
    		}
    		catch (IOException e) { 
    			logger.error(e.getMessage(), e);
	            httpResp.sendError(HttpServletResponse.SC_INTERNAL_SERVER_ERROR);
	            return;
    		}
    		
        }	
    		
	}

	@Override
	public void init(FilterConfig config) throws ServletException {
		try {
	        logger.info("Authentication is initializing.");
	        String excludedPathsStr = config.getInitParameter("excludedPaths");
	        if (excludedPathsStr != null) {
	            String[] excludedPaths = excludedPathsStr.split(",");
	            for (String exludedPath : excludedPaths) {
	                logger.debug("Add the exclude path " + exludedPath);
	                this.excludePatterns.add(exludedPath);
	            }
	        }
			} catch (Exception e) {
				//ignore
			}

	    } 

}
