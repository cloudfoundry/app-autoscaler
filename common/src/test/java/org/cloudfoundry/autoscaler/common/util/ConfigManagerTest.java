package org.cloudfoundry.autoscaler.common.util;

import static org.junit.Assert.assertEquals;

import org.cloudfoundry.autoscaler.common.util.ConfigManager;
import org.junit.Test;

public class ConfigManagerTest {
	@Test
	public void defaultValues() {
		assertEquals("NA", ConfigManager.get("doesNotExist", "NA"));
		assertEquals(1, ConfigManager.getInt("doesNotExist", 1));
	}

	@Test
	public void stringValues() {
		assertEquals("a string", ConfigManager.get("string", "fail"));
	}

	@Test
	public void intValues() {
		assertEquals(100, ConfigManager.getInt("int"));
	}
}
