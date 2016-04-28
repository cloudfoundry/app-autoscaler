package org.cloudfoundry.autoscaler.servicebroker.util;

import static org.junit.Assert.assertEquals;

import java.util.Locale;

import org.junit.Test;

/**
 *
 */
public class MessageUtilTest {

	@Test
	public void messageKeyTest() throws InterruptedException {

		assertEquals("CWSCV2004E: Another Auto-Scaling service is already bound to application.",
				MessageUtil.getMessageString("AlreadyBindedAnotherService"));
		assertEquals("CWSCV2003E: The Auto-Scaling service broker failed to bind the service instance 1.",
				MessageUtil.getMessageString("BindServiceFail", 1));
		assertEquals(
				"CWSCV2004E: \u5176\u4ed6 Auto-Scaling \u670d\u52a1\u5df2\u4e0e\u5e94\u7528\u7a0b\u5e8f\u7ed1\u5b9a\u3002",
				MessageUtil.getMessageString("AlreadyBindedAnotherService", Locale.CHINESE));
		assertEquals(
				"CWSCV2003E: Auto-Scaling \u670d\u52a1\u4ee3\u7406\u7a0b\u5e8f\u65e0\u6cd5\u7ed1\u5b9a\u670d\u52a1\u5b9e\u4f8b 1\u3002",
				MessageUtil.getMessageString("BindServiceFail", Locale.CHINESE, 1));
	}

}
