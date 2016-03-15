package org.cloudfoundry.autoscaler;

import java.util.HashMap;
import java.util.Iterator;
import java.util.Map;
import java.util.Set;

import org.apache.log4j.Logger;
import org.json.JSONArray;
import org.json.JSONObject;



@SuppressWarnings("rawtypes")
public class AutoScalerEnv {
	
	private static Map<String, String> mongoProperties = new HashMap<String, String>();
	private static final Logger logger     = Logger.getLogger(AutoScalerEnv.class.getName());
	private static String serverName = null;
	
	public static Map<String, String> getMongodbCredentials(){
        String[] propertyNames = new String[] { "host", "port", "db", "username", "password" };
        String serviceInfo = System.getenv("VCAP_SERVICES");
        logger.debug( "VCAP_SERVICES=" + serviceInfo);
		if (serviceInfo != null && !"".equals(serviceInfo)) {
			JSONObject jsonServices;

			jsonServices = new JSONObject(serviceInfo);

			@SuppressWarnings("unchecked")
			Set<String> keySet = jsonServices.keySet();
			JSONObject mongoCredentials = null;
			for (String key: keySet) {
				if (key.startsWith("mongodb")) {
					JSONArray jsonArray = (JSONArray) jsonServices.get(key);
					JSONObject jsonService = (JSONObject) jsonArray.get(0);
					mongoCredentials = (JSONObject) jsonService
							.get("credentials");
					break;
				} 
			}
			if (mongoCredentials != null) {
				for (int i = 0; i < propertyNames.length; i++) {

					String name = propertyNames[i];
					mongoProperties.put(name, mongoCredentials.get(name)
							.toString());
				}
			}
		} 
		if (mongoProperties.get("host") == null){
			//set default properties
			mongoProperties.put("host", "127.0.0.1");
			mongoProperties.put("port", "27017");
			mongoProperties.put("username", "");
			mongoProperties.put("password", "");
			mongoProperties.put("db", "autoscaling");
		}
		return mongoProperties;
	}

	
	private static String tempEnvVar; 
	public static final int    DbPort                    = ((tempEnvVar=mongoProperties.get("port"))!=null) ? Integer.parseInt(tempEnvVar) : 27017;
	public static final String DbUserName                = ((tempEnvVar=mongoProperties.get("username"))!=null) ? tempEnvVar : "";
	public static final String DbPassword                = ((tempEnvVar=mongoProperties.get("password"))!=null) ? tempEnvVar : "";
	public static final String defaultCfApiHost          = ((tempEnvVar=System.getenv("defaultCfApiHost"))!=null) ? tempEnvVar : "";
	public static final String defaultCfUser             = ((tempEnvVar=System.getenv("defaultCfUser"))!=null) ? tempEnvVar : "";
	public static final String defaultCfPassword         = ((tempEnvVar=System.getenv("defaultCfPassword"))!=null) ? tempEnvVar : "";
	private static final String VCAP_APPLICATION_ENV = "VCAP_APPLICATION";

		
	/**
	 * 
	 * @return
	 */
	public static String getApplicationUrl (){
		AppEnv env= getApplicationEnv();
		if (env == null || env.getApplication_uris().length == 0)
			return null;
		return "http://" + env.getApplication_uris()[0];
	}
	
	/**
	 * Gets host name of the server where the app runs
	 * @return
	 */
	public static String getServerName(){
		if (serverName == null){
			AppEnv env= getApplicationEnv();
			if (env == null)
				return "localhost"; //Just for test, will delete the code later
			String appUrl = env.getApplication_uris()[0];
			serverName = appUrl.substring(0, appUrl.indexOf("."));
			
		}
		return serverName;
	}
	public static AppEnv getApplicationEnv(){
		String applicationEnv = System.getenv(VCAP_APPLICATION_ENV);
		if (applicationEnv == null)
			return null;
		AppEnv appEnv = new AppEnv();
		try {
			JSONObject jsonObj = new JSONObject(applicationEnv);
			JSONArray array = (JSONArray)jsonObj.get("application_uris");
			String[] uris = new String[array.length()];
			Iterator iter = array.iterator();
			int i = 0;
			while (iter.hasNext()){
				uris[i] = iter.next().toString(); 
			}
			appEnv.setApplication_uris(uris);
		} catch (Exception e) {
			logger.error( "Error to parse application environment varables");
		}
		return appEnv;
	}


}
