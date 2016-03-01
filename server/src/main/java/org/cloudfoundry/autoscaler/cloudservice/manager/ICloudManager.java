package org.cloudfoundry.autoscaler.cloudservice.manager;


import java.util.Map;

import org.cloudfoundry.autoscaler.cloudservice.manager.exceptions.AppNotFoundException;
import org.cloudfoundry.autoscaler.cloudservice.manager.exceptions.CloudException;


public interface ICloudManager
{
	/**
	 * Checks if is logged in 
	 * @return
	 */
	public boolean isLoggedIn();

	/**
	 * Login to the cloud
	 * @param cfUser
	 * @param cfPass
	 * @param cfTarg
	 * @throws CloudException
	 */
	public void login(String cfUser, String cfPass, String cfTarg) throws CloudException;
	
	/**
	 * Login to the cloud by client_credential
	 * @param cfUser
	 * @param cfPass
	 * @param cfTarg
	 * @throws CloudException
	 */
	public void loginByClientId(String cfClientId, String cfSecretKey, String cfTarg) throws CloudException;
	
	/**
	 * Logout to the cloud
	 */
	public void logout();
	
	/**
	 * Updates instance number
	 * @param appId
	 * @param instances
	 * @throws CloudException 
	 */
	public void updateInstances(String appId, int instances) throws CloudException;
	
	/**
	 * Gets running instances
	 * @param appId
	 * @return
	 * @throws CloudException
	 */
	public int getRunningInstances(String appId) throws CloudException;
	
	/**
	 * Gets expected instances
	 * @param appId
	 * @return
	 * @throws AppNotFoundException
	 * @throws CloudException
	 */
	public int getInstances(String appId) throws AppNotFoundException, CloudException;
	
	/**
	 * Get appName
	 * @param appId
	 * @return
	 * @throws CloudException 
	 */
	public CloudApp getApplication(String appId) throws CloudException;
	
	/**
	 * Gets org and space
	 * @param appId
	 * @return
	 * @throws CloudException
	 */
	public Map<String, String> getOrgSpace(String appId) throws CloudException;
	
}
