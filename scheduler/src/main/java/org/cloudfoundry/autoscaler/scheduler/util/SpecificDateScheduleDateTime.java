package org.cloudfoundry.autoscaler.scheduler.util;


/**
 * A convenience bean to hold the schedule identifier. The schedule start date time and end date time.
 * Note: This bean is created to ease the date time validation.
 * 
 * 
 *
 */
public class SpecificDateScheduleDateTime {
	private String scheduleIdentifier;// An identifier for convenience, currently number till we come up with some other mechanism to identify schedule)to know which schedule is being processed
	private Long startDateTime;
	private Long endDateTime;
	
	public String getScheduleIdentifier() {
		return scheduleIdentifier;
	}

	public void setScheduleIdentifier(String scheduleIdentifier) {
		this.scheduleIdentifier = scheduleIdentifier;
	}

	public boolean hasValidDate() {
		return startDateTime != null && endDateTime != null;
	}

	

	public Long getStartDateTime() {
		return startDateTime;
	}

	public void setStartDateTime(Long startDateTime) {
		this.startDateTime = startDateTime;
	}

	public Long getEndDateTime() {
		return endDateTime;
	}

	public void setEndDateTime(Long endDateTime) {
		this.endDateTime = endDateTime;
	}

}
