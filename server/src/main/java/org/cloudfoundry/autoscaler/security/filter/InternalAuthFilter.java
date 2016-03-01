package org.cloudfoundry.autoscaler.security.filter;


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

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.metric.util.ConfigManager;


/**
 * Servlet Filter implementation class InternalTokenFilter
 */
@WebFilter({ "/*" })

public class InternalAuthFilter implements Filter {


    private static final String token = ConfigManager.get("internalAuthTokenBase64Encoded");

    /**
     * Default constructor.
     */
    public InternalAuthFilter() {
    }

    /**
     * @see Filter#destroy()
     */
    public void destroy() {
    }

    /**
     * @see Filter#doFilter(ServletRequest, ServletResponse, FilterChain)
     */
    public void doFilter(ServletRequest request, ServletResponse response, FilterChain chain) throws IOException,
            ServletException {

        HttpServletRequest req = (HttpServletRequest) request;
        HttpServletResponse res = (HttpServletResponse) response;
        boolean authorized = false;

        String authorization = req.getHeader("Authorization");
        if (authorization != null && authorization.startsWith("Basic")) {
            String requestToken = authorization.substring("Basic".length()).trim();
            if (token.equals(requestToken)) {
                authorized = true;
                chain.doFilter(request, response);
            }
        }
        if (!authorized) {
            res.setStatus(HttpServletResponse.SC_UNAUTHORIZED);
        }

    }

    /**
     * @see Filter#init(FilterConfig)
     */
    public void init(FilterConfig fConfig) throws ServletException {
    }

}
