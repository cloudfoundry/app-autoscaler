package org.cloudfoundry.autoscaler.servicebroker.filter;

import java.io.IOException;

import javax.servlet.Filter;
import javax.servlet.FilterChain;
import javax.servlet.FilterConfig;
import javax.servlet.ServletException;
import javax.servlet.ServletRequest;
import javax.servlet.ServletResponse;
import javax.servlet.annotation.WebFilter;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;

import org.cloudfoundry.autoscaler.servicebroker.mgr.ConfigManager;

import com.sun.jersey.core.util.Base64;


/**
 * Servlet Filter implementation class FixedTokenFilter
 */
@WebFilter({ "/v2/*" })

public class BrokerTokenAuthFilter implements Filter {

    private static final String brokerUsername = ConfigManager.get("brokerUsername");
    private static final String brokerPassword = ConfigManager.get("brokerPassword");
    private static final String brokerCredential = "Basic " + new String(Base64.encode((brokerUsername + ":" + brokerPassword).getBytes()));
	
    private static boolean checkAuthorization(HttpServletRequest httpServletRequest) {
        String authorization = httpServletRequest.getHeader("Authorization");
        if (authorization != null && authorization.startsWith("Basic")) {;
            if (authorization.equals(brokerCredential)) 
                return true;
        }
        return false;
    }
     
  /**
     * Default constructor.
     */
    public BrokerTokenAuthFilter() {
    }

    /**
     * @see Filter#destroy()
     */
    public void destroy() {
    }

    /**
     * @see Filter#init(FilterConfig)
     */
    public void init(FilterConfig fConfig) throws ServletException {
    }
    
    /**
     * @see Filter#doFilter(ServletRequest, ServletResponse, FilterChain)
     */
    public void doFilter(ServletRequest request, ServletResponse response, FilterChain chain) throws IOException,
            ServletException {
        if (!checkAuthorization((HttpServletRequest) request)) {
        	((HttpServletResponse) response).setStatus(HttpServletResponse.SC_UNAUTHORIZED);
        } else {
        	chain.doFilter(request, response);
        }
  
    }


}
