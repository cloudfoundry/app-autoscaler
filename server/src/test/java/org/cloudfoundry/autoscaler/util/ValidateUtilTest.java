package org.cloudfoundry.autoscaler.util;

import static org.junit.Assert.assertEquals;

import org.junit.Test;

public class ValidateUtilTest {
	
	@Test
	public void isNullTest(){
		assertEquals(ValidateUtil.isNull(null), true);
		assertEquals(ValidateUtil.isNull(""), true);
		assertEquals(ValidateUtil.isNull("somestr"), false);
	}

}
