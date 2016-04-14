package org.cloudfoundry.autoscaler.api.util;

import static org.junit.Assert.assertEquals;

import org.cloudfoundry.autoscaler.api.util.ConfigManager;
import org.junit.Test;

public class ConfigManagerTest {
    @Test
    public void getConfigTest() throws InterruptedException {
    	
    	//get by default value
    	assertEquals("NA", ConfigManager.get("noExisting", "NA"));
    	//get by config.file
    	assertEquals("CF-AutoScaler", ConfigManager.get("scalingServiceName", "CF-AutoScaler"));
    	
    	//get by default value, INT
    	assertEquals(1, ConfigManager.getInt("noExisting", 1));
    	//get INT
       	assertEquals(100, ConfigManager.getInt("maxMetricRecord"));
       	
       	
    }
}
