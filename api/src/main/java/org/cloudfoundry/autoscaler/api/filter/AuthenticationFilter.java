package org.cloudfoundry.autoscaler.api.filter;

import java.io.IOException;
import java.util.ArrayList;
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
import org.cloudfoundry.autoscaler.common.util.AuthenticationTool;

/**
 * Servlet Filter implementation class AuthenticationFilter
 */

public class AuthenticationFilter implements Filter {
	private static final Logger logger = Logger.getLogger(AuthenticationFilter.class.getName());
	private static final Pattern pattern = Pattern.compile("/apps/[^/]+/");
	private final ArrayList<String> excludePatterns = new ArrayList<String>();

	@Override
	public void destroy() {

	}

	@Override
	public void doFilter(ServletRequest request, ServletResponse response, FilterChain filter)
			throws IOException, ServletException {

		HttpServletResponse httpResp = (HttpServletResponse) response;
		HttpServletRequest httpReq = (HttpServletRequest) request;

		String uri = httpReq.getRequestURI().toString();
		if (excludePatterns.contains(uri)) {
			filter.doFilter(httpReq, httpResp);

		} else {

			String authorization = httpReq.getHeader("Authorization");
			if (authorization == null) {
				logger.info("No Authorization header found.");
				httpResp.setStatus(HttpServletResponse.SC_UNAUTHORIZED);
				return;
			}

			if (authorization.startsWith("Bearer "))
				authorization = authorization.replaceFirst("Bearer ", "");
			else if (authorization.startsWith("bearer "))
				authorization = authorization.replaceFirst("bearer ", "");

			String userId = null;

			try {
				userId = AuthenticationTool.getInstance().getuserIdFromToken(authorization);
				if (userId == null) {
					httpResp.setStatus(HttpServletResponse.SC_UNAUTHORIZED);
					return;
				}
			} catch (Exception e) {
				logger.error("Get exception: " + e.getClass().getName() + " with message " + e.getMessage());
				httpResp.setStatus(HttpServletResponse.SC_INTERNAL_SERVER_ERROR);
				return;
			}

			String appGuid = null;

			Matcher matcher = pattern.matcher(uri);
			if (!matcher.find()) {
				httpResp.setStatus(HttpServletResponse.SC_BAD_REQUEST);
				return;
			}

			appGuid = matcher.group(0).replace("/apps/", "").replace("/", "");
			
			logger.debug("Authentication check for appGuid " + appGuid + " with access token: " + authorization);
			try {
				boolean isDeveloper =  AuthenticationTool.getInstance().isUserDeveloperOfAppSpace(userId, authorization, appGuid);		
				if (isDeveloper) {
					logger.debug("SecurityCheck succeeded.");
					filter.doFilter(request, response);
				} else {
					logger.debug("SecurityCheck failed.");
					httpResp.setStatus(HttpServletResponse.SC_UNAUTHORIZED);
					return;
				}
			} catch (Exception e) {
				logger.error("Get exception: " + e.getClass().getName() + " with message " + e.getMessage());
				httpResp.setStatus(HttpServletResponse.SC_INTERNAL_SERVER_ERROR);
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
			// ignore
		}

	}

}
