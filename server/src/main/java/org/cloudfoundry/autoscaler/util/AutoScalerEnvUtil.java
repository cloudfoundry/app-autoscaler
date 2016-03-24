package org.cloudfoundry.autoscaler.util;

import java.util.Iterator;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.bean.AppEnv;
import org.cloudfoundry.autoscaler.constant.Constants;
import org.json.JSONArray;
import org.json.JSONObject;



@SuppressWarnings("rawtypes")
public class AutoScalerEnvUtil {
	private static final Logger logger     = Logger.getLogger(AutoScalerEnvUtil.class.getName());
	private static String serverName = null;
	

		
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
		String applicationEnv = System.getenv(Constants.VCAP_APPLICATION_ENV);
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
