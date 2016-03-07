package org.cloudfoundry.autoscaler.servicebroker.mgr;

import java.io.IOException;
import java.io.InputStream;
import java.io.UnsupportedEncodingException;
import java.net.URI;
import java.util.HashMap;
import java.util.Map;
import java.util.Properties;
import java.util.Set;

import org.apache.commons.io.IOUtils;
import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.servicebroker.Constants;
import org.cloudfoundry.autoscaler.servicebroker.exception.ProxyInitilizedFailedException;
import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;

import com.sun.jersey.core.util.Base64;


public class ConfigManager {

	private static final Logger logger = Logger.getLogger(ConfigManager.class);

    private static String CONFIG_FILE = "/config.properties";
    private static Properties props = new Properties();

    private static InputStream is = ConfigManager.class.getResourceAsStream(CONFIG_FILE);
    static {
        try {
            props.load(is);
        } catch (Exception ex) {
            logger.error(ex.getMessage());
        } finally {
            try {
                is.close();
            } catch (Exception ex) {
                logger.error(ex.getMessage());
            }
        }
    }
    

	public static JSONObject getCatalogJSON () {
		String CATALOG_FILE = "/" + get("catalogFile", "catalog.json");
		try {
			return new JSONObject(IOUtils.toString(ConfigManager.class.getResourceAsStream(CATALOG_FILE), "UTF-8"));
		} catch (IOException e) {
			logger.error ("Invalid catalog file");
		} 
		return null;
	}

    public static String get(String key, String defaultValue) {
        String v = get(key);
        if(v == null) {
            v = defaultValue;
        }
        return v;
    }

    public static String get(String key) {
        String v = System.getenv(key);
        if ((v== null) || v.isEmpty() ) {
            v = props.getProperty(key); 
        }
        return v;
    }

    public static double getDouble(String key, double defaultValue) {
        String stringValue = get(key);
        if (stringValue != null) {
            try {
                double doubleValue = Double.parseDouble(stringValue);
                return doubleValue;
            } catch (Exception e) {
                // ignore
            }
        }

        return defaultValue;
    }

    public static double getDouble(String key) {
    	return getDouble(key, 0.0);
    }

    public static int getInt(String key, int defaultValue) {
        String stringValue = get(key);
        if (stringValue != null) {
            try {
                int intValue = Integer.parseInt(stringValue);
                return intValue;
            } catch (Exception e) {
                // ignore
            }
        }

        return defaultValue;
    }

    public static int getInt(String key) {
    	return getInt(key, 0);
    }

    public static long getLong(String key, long defaultValue) {
        String stringValue = get(key);
        if (stringValue != null) {
            try {
                long longValue = Long.parseLong(stringValue);
                return longValue;
            } catch (Exception e) {
                // ignore
            }
        }

        return defaultValue;
    }

    public static long getLong(String key) {
    	return getLong(key, 0L);
    }

    public static boolean getBoolean(String key, boolean defaultValue) {

        String stringValue = get(key);
        if (stringValue != null) {
            try {
                return Boolean.parseBoolean(stringValue);
            } catch (Exception e) {
                // ignore
            }
        }
        return defaultValue;
    }    
    
	public static Boolean getBoolean(String key) {
		return getBoolean(key, false);
	}

	
    public static String getDecryptedString(String key) {
		try {
			byte[] decryptSecret = Base64.decode(ConfigManager.get(key));
			 return new String(decryptSecret,"UTF-8");
		} catch (UnsupportedEncodingException e) {
			logger.error("Fail to decrypt for " + key + " with exception " + e.getMessage());
		} 
		return null;
    }
    
    public static String getInternalAuthToken(){
    	String internalAuthUserName = ConfigManager.get("internalAuthUsername");
		String internalAuthPassword = ConfigManager.get("internalAuthPassword");
    	try {    		
			byte[] decryptSecret = Base64.encode(internalAuthUserName + ":" + internalAuthPassword);
			 return new String(decryptSecret,"UTF-8");
		} catch (UnsupportedEncodingException e) {
			logger.error("Fail to encrypt for " + internalAuthUserName + ":" + internalAuthPassword + " with exception " + e.getMessage());
		} 
		return null;
    }
    
    
	public static JSONObject parseCFEnv(String serviceName){
        JSONObject credentials = null;
        String serviceInfo = System.getenv("VCAP_SERVICES");
        try {
        	if (serviceInfo != null && !"".equals(serviceInfo)) {// in cloud
        		// foundry
        		JSONObject jsonServices = new JSONObject(serviceInfo);

        		@SuppressWarnings("unchecked")
        		Set<String> keyset = jsonServices.keySet();
        		for (String key : keyset) {
        			if (key.startsWith(serviceName)) {
        				JSONArray jsonArray = (JSONArray) jsonServices.get(key);
        				JSONObject jsonService = (JSONObject) jsonArray.get(jsonArray.length() - 1);
        				credentials = (JSONObject) jsonService.get("credentials");
        				break;
        			}
        		}
        	}		
        }   catch (JSONException e) {
        	logger.error("Incorrect JSON format");
        }   
        return credentials;
	}
	

}
