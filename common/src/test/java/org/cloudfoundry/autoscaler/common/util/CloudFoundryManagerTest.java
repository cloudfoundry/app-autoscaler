package org.cloudfoundry.autoscaler.common.util;

import org.junit.Test;

import com.sun.jersey.test.framework.JerseyTest;

import static org.junit.Assert.fail;

import org.cloudfoundry.autoscaler.common.AppNotFoundException;
import org.cloudfoundry.autoscaler.common.ClientIDLoginFailedException;
import org.cloudfoundry.autoscaler.common.CloudException;
import org.cloudfoundry.autoscaler.common.Constants;
import org.cloudfoundry.autoscaler.common.ServiceNotFoundException;
import org.cloudfoundry.autoscaler.common.util.CloudFoundryManager;
import static org.cloudfoundry.autoscaler.common.rest.mock.cc.CloudControllerRestApi.TEST_APP_ID;
import static org.cloudfoundry.autoscaler.common.rest.mock.cc.CloudControllerRestApi.TEST_SEVICE_NAME;

import org.cloudfoundry.autoscaler.common.util.ConfigManager;

public class CloudFoundryManagerTest extends JerseyTest {

	public CloudFoundryManagerTest() throws Exception {
		super("org.cloudfoundry.autoscaler.common.rest.mock.cc");
	}

	@Test
	public void testCloudFoundryLogin() {
		CloudFoundryManager manager = new CloudFoundryManager();
		try {
			manager.login();
		} catch (Exception e) {
			fail("Get exception when login");
		}
	}

	@Test(expected = ClientIDLoginFailedException.class)
	public void testCloudFoundryLoginWithNonExistClient() throws Exception {
		CloudFoundryManager manager = new CloudFoundryManager("non_exist_ID", "non_exist_secret",
				ConfigManager.get(Constants.CFURL));
		manager.login();
	}

	@Test(expected = ClientIDLoginFailedException.class)
	public void testCloudFoundryLoginWithBlankClientID() throws Exception {
		CloudFoundryManager manager = new CloudFoundryManager("", ConfigManager.get(Constants.CLIENT_SECRET),
				ConfigManager.get(Constants.CFURL));
		manager.login();
	}

	@Test(expected = ClientIDLoginFailedException.class)
	public void testCloudFoundryLoginWithBlankClientSecret() throws Exception {
		CloudFoundryManager manager = new CloudFoundryManager(ConfigManager.get(Constants.CLIENT_ID), "",
				ConfigManager.get(Constants.CFURL));
		manager.login();
	}
	
	@Test
	public void testGetServiceInfo(){
		try{
			CloudFoundryManager manager = CloudFoundryManager.getInstance();
			manager.getServiceInfo(TEST_APP_ID, TEST_SEVICE_NAME);
		}
		catch(Exception e){
			fail("Get exception when gettting service info");
		}	
	}

	@Test(expected = CloudException.class)
	public void testGetServiceInfoWithNullToken() throws Exception {
		CloudFoundryManager manager = new CloudFoundryManager();
		manager.getServiceInfo(TEST_APP_ID, TEST_SEVICE_NAME);
	}
	
	
	@Test(expected = AppNotFoundException.class)
	public void testGetServiceInfoWithNonExistAppID() throws Exception {
		CloudFoundryManager manager = CloudFoundryManager.getInstance();
		manager.getServiceInfo("non_exist_ID", TEST_SEVICE_NAME);
	}

	@Test(expected = ServiceNotFoundException.class)
	public void testGetServiceInfoWithNonExistService() throws Exception {
		CloudFoundryManager manager = CloudFoundryManager.getInstance();
		manager.getServiceInfo(TEST_APP_ID, "non_exsit_service_name");
	}

	

}
