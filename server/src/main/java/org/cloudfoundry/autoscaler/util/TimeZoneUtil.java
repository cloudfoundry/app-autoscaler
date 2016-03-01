package org.cloudfoundry.autoscaler.util;

import java.util.TimeZone;

public class TimeZoneUtil {
	
	public static TimeZone parseTimeZoneId(String timeZoneId){
		TimeZone zone = TimeZone.getDefault();
		String zoneName = "";
		if(null != timeZoneId && !"".equals(timeZoneId))
		{
			timeZoneId = timeZoneId.trim().replaceAll("\\s+", "");
			
			int index2 = timeZoneId.indexOf(")");
			if(index2 >= 0)
			{
				zoneName = timeZoneId.substring(index2+1, timeZoneId.length()).trim();
			}
			zone = TimeZone.getTimeZone(zoneName);
		}
		return zone;
	}
	public static boolean isParallel(TimeZone zone1, TimeZone zone2){
		long time = System.currentTimeMillis();
		return zone1.getOffset(time) == zone2.getOffset(time);
		
	}

}
