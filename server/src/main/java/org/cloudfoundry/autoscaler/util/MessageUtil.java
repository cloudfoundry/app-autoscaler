
package org.cloudfoundry.autoscaler.util;

import java.text.MessageFormat;
import java.util.Locale;
import java.util.ResourceBundle;

/**
 * Resouce bundle messages util
 * 
 * 
 * 
 */
public class MessageUtil {

	private final static String BUNDLE_NAME="org.cloudfoundry.autoscaler.nls.ServerMessages";
	
    /**
     * Gets messages
     * 
     * @param key
     * @return
     */
	public static String getMessageString(String key) {
        return getMessageString(key, new Object[]{}); 
    }

    /**
     * Get messages
     * 
     * @param key
     * @param params
     * @return
     */
    public static String getMessageString(String key, Object... params) {
		ResourceBundle NONLS_MESSAGE_RESOURCE_BUNDLE = ResourceBundle.getBundle(BUNDLE_NAME);
        if (params == null || params.length == 0) 
            return NONLS_MESSAGE_RESOURCE_BUNDLE.getString(key);
        else
        	return MessageFormat.format(NONLS_MESSAGE_RESOURCE_BUNDLE.getString(key), params);
    }
    
    /**
     * Gets messages
     * 
     * @param key
     * @return
     */
	public static String getMessageString(String key, Locale locale) {
		return getMessageString(key, locale, new Object[]{}); 
    }    
    
    /**
     * Get messages
     * 
     * @param key
     * @param locale
     * @param params
     * @return
     */
    public static String getMessageString(String key, Locale locale, Object... params) {
    	ResourceBundle MESSAGE_RESOURCE_BUNDLE = ResourceBundle.getBundle(BUNDLE_NAME, locale); //$NLS$
    	if (params == null || params.length == 0) 
            return MESSAGE_RESOURCE_BUNDLE.getString(key);
        else
        	return MessageFormat.format(MESSAGE_RESOURCE_BUNDLE.getString(key), params);
   
    }    
    
}
