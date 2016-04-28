package org.cloudfoundry.autoscaler.schedule;

import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.Calendar;
import java.util.Date;
import java.util.List;
import java.util.Map;
import java.util.TimeZone;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.common.util.TimeZoneUtil;
import org.cloudfoundry.autoscaler.data.couchdb.document.Application;
import org.cloudfoundry.autoscaler.data.couchdb.document.AutoScalerPolicy;
import org.cloudfoundry.autoscaler.data.couchdb.document.ScheduledPolicy;
import org.cloudfoundry.autoscaler.data.couchdb.document.ScheduledPolicy.ScheduledType;
import org.cloudfoundry.autoscaler.exceptions.DataStoreException;
import org.cloudfoundry.autoscaler.exceptions.PolicyNotFoundException;
import org.cloudfoundry.autoscaler.manager.ApplicationManager;
import org.cloudfoundry.autoscaler.manager.ApplicationManagerImpl;
import org.cloudfoundry.autoscaler.manager.PolicyManager;
import org.cloudfoundry.autoscaler.manager.PolicyManagerImpl;

public class ScheduledServiceUtil {
	private static final String CLASS_NAME = ScheduledServiceUtil.class.getName();
	private static final Logger logger  = Logger.getLogger(CLASS_NAME);
	public static void updateScheduledPolicyBasedOnTime(AutoScalerPolicy autoScalerPolicy) throws ParseException, PolicyNotFoundException, DataStoreException {
		String toSetScheduledId = null;
		Date current = new Date();
		
		for(Map.Entry<String, ScheduledPolicy> scheduledPolicyEntry: autoScalerPolicy.getScheduledPolicies().entrySet()) {
			String scheduledPolicyId = scheduledPolicyEntry.getKey();
			ScheduledPolicy scheduledPolicy = scheduledPolicyEntry.getValue();
			String scheduledType = scheduledPolicy.getType();
			String timezoneId = scheduledPolicy.getTimezone();
			TimeZone curTimeZone = TimeZone.getDefault();
			TimeZone policyTimeZone = TimeZone.getDefault();
			policyTimeZone = TimeZoneUtil.parseTimeZoneId(timezoneId);
			if (ScheduledType.RECURRING.name().equals(scheduledType)) {
				Date startTime = new SimpleDateFormat(ScheduledPolicy.recurringDateFormat).parse(scheduledPolicy.getStartTime());				
				Date endTime = new SimpleDateFormat(ScheduledPolicy.recurringDateFormat).parse(scheduledPolicy.getEndTime());				
				startTime = new Date(startTime.getTime() - policyTimeZone.getOffset(System.currentTimeMillis()) + curTimeZone.getOffset(System.currentTimeMillis()));
				endTime = new Date(endTime.getTime() - policyTimeZone.getOffset(System.currentTimeMillis()) + curTimeZone.getOffset(System.currentTimeMillis()));
				Date startTimewithDay = generateTime(current, 0, startTime);
				Date endTimewithDay = generateTime(current, 0, endTime);
				String repeatCycle = scheduledPolicy.getRepeatCycle();
				Date date1 = new Date(System.currentTimeMillis() + policyTimeZone.getOffset(System.currentTimeMillis()) - curTimeZone.getOffset(System.currentTimeMillis()));
				String dayOfWeek = String.valueOf(dayOfWeek(date1));
				logger.debug("Schedule Type :" + scheduledType + "schedule id: "+ scheduledPolicyId + " toSetScheduledId:" + toSetScheduledId + " endTime in cur zone :" + endTimewithDay +" startTime in cur zone :" + startTimewithDay + " repeat cyble:" + repeatCycle + " dayOfWeek:" + dayOfWeek);
				int flag = repeatCycle.indexOf(dayOfWeek);
				if (flag >= 0)
				{
					if(startTimewithDay.compareTo(endTimewithDay) > 0)
					{
						if((startTimewithDay.compareTo(current) <= 0 || endTimewithDay.compareTo(current) >= 0))
						{
							if (toSetScheduledId == null) {
								logger.debug("***Schedule Type :" + scheduledType + "schedule id: "+ scheduledPolicyId + " toSetScheduledId:" + toSetScheduledId);
								toSetScheduledId = scheduledPolicyId;
							}
						}
					}
					else
					{
						if((startTimewithDay.compareTo(current) <= 0 && endTimewithDay.compareTo(current) >= 0))
						{
							if (toSetScheduledId == null) {
								logger.debug("***Schedule Type :" + scheduledType + "schedule id: "+ scheduledPolicyId + " toSetScheduledId:" + toSetScheduledId);
								toSetScheduledId = scheduledPolicyId;
							}
						}
					}
				}
				
			} else if (ScheduledType.SPECIALDATE.toString().equals(scheduledType)) {
				Date startTime = new SimpleDateFormat(ScheduledPolicy.specialDateDateFormat).parse(scheduledPolicy.getStartTime());
				Date endTime = new SimpleDateFormat(ScheduledPolicy.specialDateDateFormat).parse(scheduledPolicy.getEndTime());
				logger.debug("Schedule Type :" + scheduledType + " startTime in policy :" + startTime);
				
				startTime = new Date(startTime.getTime() - policyTimeZone.getOffset(startTime.getTime()) + curTimeZone.getOffset(System.currentTimeMillis()));
				endTime = new Date(endTime.getTime() - policyTimeZone.getOffset(endTime.getTime()) + curTimeZone.getOffset(System.currentTimeMillis()));
				
				logger.debug("Schedule Type :" + scheduledType + "schedule id: "+ scheduledPolicyId +" startTime in cur zone :" + startTime);
				if (startTime.compareTo(current) <= 0 && endTime.compareTo(current) >= 0) {
					toSetScheduledId = scheduledPolicyId;
				}
			}
		}
		logger.debug(" ***toSetScheduledId:" + toSetScheduledId);
		if ((toSetScheduledId == null && autoScalerPolicy
				.getCurrentScheduledPolicyId() != null)
				|| (toSetScheduledId != null && !toSetScheduledId
						.equals(autoScalerPolicy.getCurrentScheduledPolicyId()))) {
			logger.debug(" *** real toSetScheduledId:" + toSetScheduledId);
			PolicyManager policyManager = PolicyManagerImpl.getInstance();
			AutoScalerPolicy policyFromDB = policyManager
					.getPolicyById(autoScalerPolicy.getId());
			policyFromDB.setCurrentScheduledPolicyId(toSetScheduledId);
			policyManager.updatePolicy(policyFromDB);

			ApplicationManager appManager = ApplicationManagerImpl
					.getInstance();
			List<Application> apps = appManager
					.getApplicationByPolicyId(autoScalerPolicy.getPolicyId());
			for (Application app : apps) {
				try {
					    if(null != app.getPolicyState() && AutoScalerPolicy.STATE_ENABLED.equals(app.getPolicyState())){
					    	logger.debug("App " + app.getAppId() + "'s policy state is " + app.getPolicyState() + ". It will be scheduled.");
					    	appManager.handleInstancesByPolicy(app.getAppId(), policyFromDB);
					    	
					    }
					    else
					    {
					    	logger.debug("App " + app.getAppId() + "'s policy state is " + app.getPolicyState() + ". It will not be scheduled.");
					    }
					    
				} catch (Exception e) {
				}
			}
			
		}
	}
	
	public static int dayOfWeek(Date time) {
		Calendar c = Calendar.getInstance();
		c.setTime(time);
		int dayForWeek = 0;
		if (c.get(Calendar.DAY_OF_WEEK) == 1) {
			dayForWeek = 7;
		} else {
			dayForWeek = c.get(Calendar.DAY_OF_WEEK) - 1;
		}
		return dayForWeek;
	}
	
	public static Date generateTime(Date startDay, int daycount, Date timeInDay) {
		Calendar resultCalendar = Calendar.getInstance();
		Calendar timeInDayCal = Calendar.getInstance();

		Date generated = new Date(startDay.getTime() + 24 * 60 * 60 * 1000L * daycount );
		resultCalendar.setTime(generated);
		timeInDayCal.setTime(timeInDay);
		resultCalendar.set(Calendar.HOUR_OF_DAY, timeInDayCal.get(Calendar.HOUR_OF_DAY));
		resultCalendar.set(Calendar.MINUTE, timeInDayCal.get(Calendar.MINUTE));
		resultCalendar.set(Calendar.SECOND, timeInDayCal.get(Calendar.SECOND));
		return resultCalendar.getTime();
	}
}
