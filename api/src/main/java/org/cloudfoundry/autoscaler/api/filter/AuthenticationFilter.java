package org.cloudfoundry.autoscaler.api.filter;

import java.io.IOException;
import java.util.ArrayList;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import java.util.Map;

import javax.servlet.Filter;
import javax.servlet.FilterChain;
import javax.servlet.FilterConfig;
import javax.servlet.ServletException;
import javax.servlet.ServletRequest;
import javax.servlet.ServletResponse;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;

import org.apache.log4j.Logger;

import org.cloudfoundry.autoscaler.api.util.CloudFoundryManager;
import org.cloudfoundry.autoscaler.api.util.SecurityCheckStatus;
import org.cloudfoundry.autoscaler.api.util.AuthenticationTool;
import org.cloudfoundry.autoscaler.api.util.LocaleUtil;
import org.cloudfoundry.autoscaler.api.util.RestApiResponseHandler;
import org.cloudfoundry.autoscaler.api.exceptions.AppNotFoundException;
import org.cloudfoundry.autoscaler.api.exceptions.AppInfoNotFoundException;
/**
 * Servlet Filter implementation class SSOFilter
 */

public class AuthenticationFilter implements Filter {
    private static final Logger logger = Logger.getLogger(AuthenticationFilter.class.getName());
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
            // just pass down the filter chain.
            filter.doFilter(httpReq, httpResp);

        } else {
            String orgGuid=null;
            String spaceGuid=null;
         
            String appGuid = null;
            String regex = "/apps/[^/]+/";    		
    		Pattern p = Pattern.compile(regex);
    		Matcher matcher = p.matcher(uri);
    		if (matcher.find()){
    			appGuid = matcher.group(0).replace("/apps/", "").replace("/", "");
    			try {
           	        Map <String, String> orgSpace = CloudFoundryManager.getInstance().getOrgSpaceByAppId(appGuid);
        	        orgGuid = orgSpace.get("ORG_GUID");
        	        spaceGuid = orgSpace.get("SPACE_GUID");
        		}
    			catch (AppNotFoundException e){
    				HttpServletRequest httpServletRequest = (HttpServletRequest)request;
    				HttpServletResponse resp = (HttpServletResponse)response;
    				resp.sendError(HttpServletResponse.SC_NOT_FOUND, RestApiResponseHandler.getErrorMessage(e, LocaleUtil.getLocale(httpServletRequest)));
    				return;
    			}
    			catch (AppInfoNotFoundException e){
    				HttpServletRequest httpServletRequest = (HttpServletRequest)request;
    				HttpServletResponse resp = (HttpServletResponse)response;
    				resp.sendError(HttpServletResponse.SC_BAD_REQUEST, RestApiResponseHandler.getErrorMessage(new AppNotFoundException(e.getAppId()), LocaleUtil.getLocale(httpServletRequest)));
    				return;
    			}
    			catch (Exception e){
        			//throw new ServletException("appId=" + appGuid + " " + e.getMessage());
        			logger.error("Get exception: " + e.getClass().getName() + " with message " + e.getMessage());
        			HttpServletResponse resp = (HttpServletResponse)response;
                    resp.sendError(HttpServletResponse.SC_UNAUTHORIZED);
                    return;
        		}
    		}
    		
    		
            String authorization = httpReq.getHeader("Authorization");
            try {
            	logger.debug("Authentication check with access token: " + authorization);
            	logger.debug("Authentication check with appGuid: " + appGuid );
                

                SecurityCheckStatus securityCheckStatus =  null;
                if (authorization != null ) {
                	if (authorization.startsWith("Bearer "))
                		authorization = authorization.replaceFirst("Bearer ", "");
                	else if (authorization.startsWith("bearer "))
                		authorization = authorization.replaceFirst("bearer ", "");
                    securityCheckStatus = AuthenticationTool.getInstance().doValidateToken(httpReq, httpResp, authorization, orgGuid, spaceGuid);
                }
                else{
                	//securityCheckStatus = SecurityCheckStatus.SECURITY_CHECK_ERROR;
                	throw new ServletException("Current user has no permission in current space.");
                }
               	
                logger.debug("Authentication status result: " + securityCheckStatus.toString());
                if (securityCheckStatus == SecurityCheckStatus.SECURITY_CHECK_COMPLETE) {
                    logger.debug("SSO doSecurityCheck succeeded.");
                    filter.doFilter(request, response);
                }
            } catch (ServletException e) {
                logger.error(e.getMessage(), e);
                HttpServletResponse resp = (HttpServletResponse)response;
                resp.sendError(HttpServletResponse.SC_UNAUTHORIZED);
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
