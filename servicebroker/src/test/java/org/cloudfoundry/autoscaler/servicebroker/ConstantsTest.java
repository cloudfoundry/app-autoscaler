package org.cloudfoundry.autoscaler.servicebroker;

import static org.junit.Assert.assertEquals;

import org.cloudfoundry.autoscaler.servicebroker.Constants.MESSAGE_KEY;
import org.junit.Test;


/**
 *
 */
public class ConstantsTest {

    @Test
    public void messageKeyTest() throws InterruptedException {
    	
    	//error code in use:
    	assertEquals(499, MESSAGE_KEY.BindServiceFail.getErrorCode());
    	//default error code:
    	assertEquals(500, MESSAGE_KEY.QueryServiceEnabledInfoFail.getErrorCode());
    }

  
}
