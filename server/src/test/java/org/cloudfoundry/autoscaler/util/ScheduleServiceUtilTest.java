package org.cloudfoundry.autoscaler.util;

import static org.junit.Assert.assertEquals;

import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.Calendar;
import java.util.Date;

import org.cloudfoundry.autoscaler.schedule.ScheduledServiceUtil;
import org.junit.Test;

public class ScheduleServiceUtilTest {
	
	@Test
	public void dayOfWeekTest(){
		
		Calendar c = Calendar.getInstance();
		c.set(Calendar.DAY_OF_WEEK, 1);
		Date date = new Date(c.getTimeInMillis());
		assertEquals(ScheduledServiceUtil.dayOfWeek(date),  7);
		c.set(Calendar.DAY_OF_WEEK, 3);
		date = new Date(c.getTimeInMillis());
		assertEquals(ScheduledServiceUtil.dayOfWeek(date),  2);
	}
	@Test
	public void generateTimeTest() throws ParseException{
		int year = 2016;
		int month = 1;
		int day = 1;
		String timeStr = "12:00:00";
		SimpleDateFormat format = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss");
		Date date = format.parse(String.valueOf(year) + "-" + String.valueOf(month) + "-" + String.valueOf(day + 2) + " " + timeStr);
		SimpleDateFormat format2 = new SimpleDateFormat("HH:mm:ss");
		Date date2 = format.parse(String.valueOf(year) + "-" + String.valueOf(month) + "-" + String.valueOf(day) + " " + timeStr);
		Date date3 = format2.parse(timeStr);
		assertEquals(ScheduledServiceUtil.generateTime(date2, 2, date3),date);
		
	}

}
