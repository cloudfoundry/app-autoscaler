package org.cloudfoundry.autoscaler.common;

import static org.junit.Assert.assertEquals;

import org.cloudfoundry.autoscaler.common.Constants;
import org.junit.Test;

/**
 *
 */
public class ConstantsTest {

	@Test
	public void ContantKeyTest() throws InterruptedException {
		assertEquals(300, Constants.getTriggerDefaultInt("statWindow"));
		assertEquals(600, Constants.getTriggerDefaultInt("breachDuration"));
		assertEquals(600, Constants.getTriggerDefaultInt("stepDownCoolDownSecs"));
		assertEquals(600, Constants.getTriggerDefaultInt("stepUpCoolDownSecs"));
		assertEquals(30, Constants.getTriggerDefaultInt("lowerThreshold"));
		assertEquals(80, Constants.getTriggerDefaultInt("upperThreshold"));
		assertEquals(1, Constants.getTriggerDefaultInt("instanceStepCountDown"));
		assertEquals(1, Constants.getTriggerDefaultInt("instanceStepCountUp"));

		String[] metrics = Constants.getMetricTypeByAppType("java");
		assertEquals(1, metrics.length);
		assertEquals("Memory", metrics[0]);

		metrics = Constants.getMetricTypeByAppType("nodejs");
		assertEquals(1, metrics.length);
		assertEquals("Memory", metrics[0]);
	}
}
