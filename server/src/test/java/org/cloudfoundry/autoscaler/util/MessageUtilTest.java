package org.cloudfoundry.autoscaler.util;

import static org.junit.Assert.assertEquals;

import java.util.Locale;

import org.cloudfoundry.autoscaler.util.MessageUtil;
import org.junit.Test;

public class MessageUtilTest {
	
    @Test
	public void getMessageStringTest() {
    	assertEquals(MessageUtil.getMessageString("RestResponseErrorMsg_app_not_found_error"), "CWSCV3006E: The application is not found: {0}");
    	assertEquals(MessageUtil.getMessageString("RestResponseErrorMsg_app_not_found_error", Locale.GERMAN), "CWSCV3006E: Die Anwendung wurde nicht gefunden: {0}");
    }
    
}
