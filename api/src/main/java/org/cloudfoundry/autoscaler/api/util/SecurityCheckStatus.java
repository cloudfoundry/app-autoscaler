package org.cloudfoundry.autoscaler.api.util;

public enum SecurityCheckStatus {
    SECURITY_CHECK_SSO,        //internal state, request will be redirected to somewhere else. 
    SECURITY_CHECK_COMPLETE,    //final state, check is successful and user data being written into session. 
    SECURITY_CHECK_ERROR        //final state, request is redirected to error page.
}
