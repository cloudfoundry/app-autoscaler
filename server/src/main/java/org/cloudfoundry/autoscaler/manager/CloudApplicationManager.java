package org.cloudfoundry.autoscaler.manager;

import java.util.Map;

import org.cloudfoundry.autoscaler.common.util.CloudFoundryManager;

public class CloudApplicationManager {
	public final static CloudApplicationManager cloudApplicationManager = new CloudApplicationManager();
	
	private CloudApplicationManager(){
		
	}
	
	public static CloudApplicationManager getInstance() {
		return cloudApplicationManager;
	}
	/**
	 * Scale in or scale out application
	 * @param appId
	 * @param targetInstances
	 * @throws Exception 
	 */
	public void scaleApplication(String appId, int targetInstances) throws Exception {
		CloudFoundryManager.getInstance().updateInstances(appId, targetInstances);	
	}
	
	/**
	 * Gets instance number
	 * @param appId
	 * @return
	 * @throws Exception 
	 */
	public int getInstances(String appId) throws Exception{
		return CloudFoundryManager.getInstance().getAppInstancesByAppId(appId);
	}
	
	/**
	 * Gets running instance number
	 * @param appId
	 * @return
	 * @throws Exception 
	 * @throws  
	 */
	public int getRunningInstances(String appId) throws Exception{
		return CloudFoundryManager.getInstance().getRunningInstances(appId);
	}
	
	/**
	 * Gets org and space
	 * @param appId
	 * @return
	 * @throws Exception 
	 */
	public Map<String, String> getOrgSpace(String appId) throws Exception{
		return CloudFoundryManager.getInstance().getOrgSpaceByAppId(appId);
	}

}
