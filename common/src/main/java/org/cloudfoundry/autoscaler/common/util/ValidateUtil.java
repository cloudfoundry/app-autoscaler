package org.cloudfoundry.autoscaler.common.util;

public class ValidateUtil {
	public static boolean isNull(String str){
		if (str == null || str.trim().length() == 0)
			return true;
		return false;
	}
}
