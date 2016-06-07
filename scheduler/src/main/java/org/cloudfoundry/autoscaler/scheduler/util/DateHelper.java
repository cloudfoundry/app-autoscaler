package org.cloudfoundry.autoscaler.scheduler.util;

import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.TimeZone;

/**
 * @author Fujitsu
 *
 */
public class DateHelper {

	public static Date getDateWithZoneOffset(long dateTime, TimeZone policyTimeZone) {

		TimeZone currentTimeZone = TimeZone.getDefault();

		Date date = new Date(dateTime - policyTimeZone.getRawOffset() + currentTimeZone.getRawOffset());

		return date;
	}

	public static String convertDateToString(Date date) {
		SimpleDateFormat sdf = new SimpleDateFormat("yyyy-MM-dd");

		return sdf.format(date);
	}

	public static String convertTimeToString(Date date) {
		SimpleDateFormat sdf = new SimpleDateFormat("HH:mm:ss");

		return sdf.format(date);
	}

}
