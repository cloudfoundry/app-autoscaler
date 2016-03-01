package org.cloudfoundry.autoscaler.servicebroker.util;

import java.text.MessageFormat;
import java.util.Locale;
import java.util.ResourceBundle;

public class MessageUtil {

	private final static String BUNDLE_NAME="org.cloudfoundry.autoscaler.servicebroker.nls.messages";
	private final static ResourceBundle NONLS_MESSAGE_RESOURCE_BUNDLE = ResourceBundle
            .getBundle(BUNDLE_NAME); //$NON-NLS-1$
	
   /**
     * Gets messages
     * 
     * @param key
     * @return
     */
	public static String getMessageString(String key) {
        return NONLS_MESSAGE_RESOURCE_BUNDLE.getString(key);
    }

    /**
     * Get messages
     * 
     * @param key
     * @param params
     * @return
     */
    public static String getMessageString(String key, Object... params) {
        if (params == null || params.length == 0) {
            getMessageString(key);
        }
        return MessageFormat.format(NONLS_MESSAGE_RESOURCE_BUNDLE.getString(key), params);
    }
    
    /**
     * Gets messages
     * 
     * @param key
     * @return
     */
	public static String getMessageString(String key, Locale locale) {
    	ResourceBundle MESSAGE_RESOURCE_BUNDLE = ResourceBundle
                .getBundle(BUNDLE_NAME, locale); //$NON-NLS-1$
        return MESSAGE_RESOURCE_BUNDLE.getString(key);
    }    
    
    /**
     * Get messages with locale
     * 
     * @param key
     * @param locale
     * @param params
     * @return
     */
    public static String getMessageString(String key, Locale locale, Object... params) {
        if (params == null || params.length == 0) {
            return getMessageString(key,locale);
        }  
    	ResourceBundle MESSAGE_RESOURCE_BUNDLE = ResourceBundle
                .getBundle(BUNDLE_NAME, locale); //$NON-NLS-1$
        return MessageFormat.format(MESSAGE_RESOURCE_BUNDLE.getString(key), params);
    }    
}
