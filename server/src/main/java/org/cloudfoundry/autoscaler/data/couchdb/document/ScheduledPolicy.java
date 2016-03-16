package org.cloudfoundry.autoscaler.data.couchdb.document;

public class ScheduledPolicy {

	
	public static enum ScheduledType{
		RECURRING, SPECIALDATE
	}
	
	public final static String recurringDateFormat = "HH:mm";  
	public final static String specialDateDateFormat = "yyyy-MM-dd HH:mm";  
	
	private int instanceMinCount;
	private int instanceMaxCount;
	private String startTime;
	private String endTime;
	private String type;
	private String repeatCycle;
	private String timezone;
	
	public int getInstanceMinCount() {
		return instanceMinCount;
	}
	public void setInstanceMinCount(int instanceMinCount) {
		this.instanceMinCount = instanceMinCount;
	}
	public int getInstanceMaxCount() {
		return instanceMaxCount;
	}
	public void setInstanceMaxCount(int instanceMaxCount) {
		this.instanceMaxCount = instanceMaxCount;
	}
	public String getStartTime() {
		return startTime;
	}
	public void setStartTime(String startTime) {
		this.startTime = startTime;
	}
	public String getEndTime() {
		return endTime;
	}
	public void setEndTime(String endTime) {
		this.endTime = endTime;
	}
	public String getType() {
		return type;
	}
	public void setType(String type) {
		this.type = type;
	}
	public String getRepeatCycle() {
		return repeatCycle;
	}
	public void setRepeatCycle(String repeatCycle) {
		this.repeatCycle = repeatCycle;
	}
	public String getTimezone() {
		return timezone;
	}
	public void setTimezone(String timezone) {
		this.timezone = timezone;
	}
}
