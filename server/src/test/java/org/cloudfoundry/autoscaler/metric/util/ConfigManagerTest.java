package org.cloudfoundry.autoscaler.metric.util;

import static org.junit.Assert.assertEquals;

import org.cloudfoundry.autoscaler.common.util.ConfigManager;
import org.junit.Test;

public class ConfigManagerTest {
	
	@Test
    public void getTest() throws InterruptedException {
    	
    	//get by default value
    	assertEquals("NA", ConfigManager.get("noExisting", "NA"));
    	//get by config.file
    	assertEquals("continuously", ConfigManager.get("couchdbMetricDBRolloutFrequency"));
    	
    	//get by default value, INT
    	assertEquals(1, ConfigManager.getInt("noExisting", 1));
    	//get INT
       	assertEquals(10000, ConfigManager.getInt("couchdbTimeout"));
       	
    	//get by default value, Long
    	assertEquals(1L, ConfigManager.getLong("noExisting", 1));
    	//get by system.env Long
       	assertEquals(10000L, ConfigManager.getLong("couchdbTimeout"));
       	
      //get by default value, Double
    	assertEquals(1, ConfigManager.getDouble("noExisting", 1),0);
    	//get by system.env Double
       	assertEquals(10000, ConfigManager.getDouble("couchdbTimeout"),0);

    	//get by default value, boolean
    	assertEquals(false, ConfigManager.getBoolean("noExisting", false));
       	
    }

}
