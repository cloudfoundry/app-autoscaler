package org.cloudfoundry.autoscaler.data.couchdb;

import java.text.SimpleDateFormat;
import java.util.Calendar;
import java.util.HashMap;
import java.util.TimeZone;

import org.cloudfoundry.autoscaler.common.util.ConfigManager;
import org.cloudfoundry.autoscaler.constant.Constants;
import org.cloudfoundry.autoscaler.data.couchdb.document.MetricDBSegment;

public class MetricDBSegmentManager {

    private static final String rolloutFrequency = ConfigManager.get("couchdbMetricDBRolloutFrequency");
	private static final int rolloutFrequencyCustomValue = 5;
	private static final TimeZone UTC = TimeZone.getTimeZone("UTC"); 

    private static volatile MetricDBSegmentManager instance = null;
    private static HashMap<String, MetricDBSegment> metricDBSegmentMap = new HashMap<String, MetricDBSegment>();
	
    public static MetricDBSegmentManager getInstance(){
        if (instance == null) {
        	synchronized (MetricDBSegmentManager.class) {
        		if (instance == null) 
        			instance = new MetricDBSegmentManager();
        	}
        }    	return instance;
    }
    
    private String generateWeeklyPostfix (Calendar now){
    	return "-" + now.get(Calendar.WEEK_OF_MONTH);
    }
    
    private String generateDailyPostfix (Calendar now){
    	return "-" + now.get(Calendar.DAY_OF_WEEK);
    }
    
    private String generateHourlyPostfix (Calendar now){
    	return "-" + now.get(Calendar.HOUR_OF_DAY);
    }
    
	private void tuneToEndOfDay(Calendar now){
		now.set(Calendar.HOUR_OF_DAY, 23);
		now.set(Calendar.MINUTE, 59);
		now.set(Calendar.SECOND, 59);
		now.set(Calendar.MILLISECOND, 999);
	}
	
	private long calculateEndTimestampByMonth(Calendar now){
		Calendar lastDate = (Calendar) now.clone();
		lastDate.set(Calendar.DATE,1);
		lastDate.add(Calendar.MONTH,1);
		lastDate.add(Calendar.DATE,-1);
		tuneToEndOfDay(lastDate); 
		return lastDate.getTimeInMillis();
	}

	private long calculateEndTimestampByWeek(Calendar now){
		Calendar lastDate = (Calendar) now.clone();
		lastDate.set(Calendar.DAY_OF_WEEK,1);
		lastDate.add(Calendar.WEEK_OF_MONTH,1);
		lastDate.add(Calendar.DATE,-1);
		tuneToEndOfDay(lastDate); 
		return lastDate.getTimeInMillis();
	}

	private long calculateEndTimestampByDay(Calendar now){
		Calendar lastDate = (Calendar) now.clone();
		tuneToEndOfDay(lastDate); 
		return lastDate.getTimeInMillis();
	}
	
	private long calculateEndTimestampByHour(Calendar now){
		Calendar lastDate = (Calendar) now.clone();
		lastDate.get(Calendar.HOUR_OF_DAY);
		lastDate.set(Calendar.MINUTE, 59);
		lastDate.set(Calendar.SECOND, 59);
		lastDate.set(Calendar.MILLISECOND, 999);
		return lastDate.getTimeInMillis();
	}
    
	
	private long calculateEndTimestampByMinute(Calendar now, int minutes){
		Calendar lastDate = (Calendar) now.clone();
		lastDate.get(Calendar.HOUR_OF_DAY);
		lastDate.add(Calendar.MINUTE, minutes);
		lastDate.set(Calendar.SECOND, 59);
		lastDate.set(Calendar.MILLISECOND, 999);
		return lastDate.getTimeInMillis();
	}
    
    
    public MetricDBSegment getMetricDBSegment(Calendar now, String serverName) {

    	//calculate everything based on UTC
    	now.setTimeZone(UTC);
    	
    	//set segment with the default rolloutFrequency "monthly" when missing valid setting.
    	MetricDBSegment segment = new MetricDBSegment();
    	String metricDBPostfix =  now.get(Calendar.YEAR) + "-" +	 (now.get(Calendar.MONTH)+1) ; 
		long startTimestamp = now.getTimeInMillis();
		long endTimestamp = this.calculateEndTimestampByMonth(now);
    	
		if (rolloutFrequency.equalsIgnoreCase(Constants.MONTHLY)) {
			//keep the monthly as default setting
		}else if (rolloutFrequency.equalsIgnoreCase(Constants.WEEKLY)) {
			metricDBPostfix +=  generateWeeklyPostfix(now);
			endTimestamp = this.calculateEndTimestampByWeek(now);
		} else if (rolloutFrequency.equalsIgnoreCase(Constants.DAILY)) {
			metricDBPostfix +=  generateWeeklyPostfix(now) + generateDailyPostfix(now);
			endTimestamp = this.calculateEndTimestampByDay(now);
		} else if (rolloutFrequency.equalsIgnoreCase(Constants.HOURLY)) {
			metricDBPostfix +=  generateWeeklyPostfix(now) + generateDailyPostfix(now) + generateHourlyPostfix(now);
			endTimestamp = this.calculateEndTimestampByHour(now);
		} else if (rolloutFrequency.equalsIgnoreCase(Constants.CUSTOM)){
			metricDBPostfix +=  generateWeeklyPostfix(now) + generateDailyPostfix(now) + generateHourlyPostfix(now) + "-"  + now.get(Calendar.MINUTE);
			endTimestamp = this.calculateEndTimestampByMinute(now, rolloutFrequencyCustomValue);
		} else if (rolloutFrequency.equalsIgnoreCase(Constants.CONTINUOUS)) {
			metricDBPostfix =  "continuously";
			endTimestamp = Long.MAX_VALUE;	 //forever
		}

		segment.setMetricDBPostfix(metricDBPostfix);
		segment.setStartTimestamp(startTimestamp);
		segment.setEndTimestamp(endTimestamp);
		segment.setServerName(serverName);
		
		SimpleDateFormat sdf=new SimpleDateFormat("yyyy-MM-dd HH:mm:ss:S");
		sdf.setTimeZone(TimeZone.getTimeZone("UTC"));
		
		return segment;
    }

	public HashMap<String, MetricDBSegment> getMetricDBSegmentMap() {
		return metricDBSegmentMap;
	}
    
	public void addToMetricDBSegmentMap(MetricDBSegment segment) {
		metricDBSegmentMap.put(segment.getMetricDBPostfix(), segment);
	}
    
	
    
    

}
