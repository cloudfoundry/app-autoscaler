package org.cloudfoundry.autoscaler.metric.util;

import java.io.IOException;
import java.util.HashMap;
import java.util.Map;

import org.apache.log4j.Logger;

import com.sun.jersey.core.util.Base64;


public class DBAccessInfoManager {
	
	private static final Logger logger = Logger.getLogger(DBAccessInfoManager.class);
    private static volatile DBAccessInfoManager instance;
    
    private String username;
    private String password;
    private String host;
    private int port;
    	
    private DBAccessInfoManager() {
    	getCouchdbAccessInfo();
    }

    public static DBAccessInfoManager getInstance() {
        if (instance == null) {
        	synchronized (DBAccessInfoManager.class) {
        		if (instance == null) 
        			instance = new DBAccessInfoManager();
        	}
        }
        return instance;
    }

	public String getUsername() {
		return username;
	}

	public void setUsername(String username) {
		this.username = username;
	}

	public String getPassword() {
		return password;
	}

	public void setPassword(String password) {
		this.password = password;
	}

	public String getHost() {
		return host;
	}

	public void setHost(String host) {
		this.host = host;
	}

	public int getPort() {
		return port;
	}

	public void setPort(int port) {
		this.port = port;
	}
	
	

	private void getCouchdbAccessInfo(){
		Map<String, String> couchdbProperties = new HashMap<String, String>();

		try {

			byte[] decryptSecret = Base64.decode(ConfigManager.get("couchdbPasswordBase64Encoded"));
			String couchdbPwd = new String(decryptSecret, "UTF-8");
			couchdbProperties.put("password", couchdbPwd);
			couchdbProperties.put("username", ConfigManager.get("couchdbUsername"));
			couchdbProperties.put("host", ConfigManager.get("couchdbHost"));
			couchdbProperties.put("port", ConfigManager.get("couchdbPort"));

		} catch (IOException e) {
			logger.error("Incorrect JSON format");
		}

		username = (String) couchdbProperties.get("username");
		password = (String) couchdbProperties.get("password");
		host = (String) couchdbProperties.get("host");
		port = Integer.parseInt(couchdbProperties.get("port"));

    	
    }
    
    
}
