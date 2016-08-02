package org.cloudfoundry.autoscaler.scheduler.util;

import java.util.Date;

/**
 * A convenience bean to hold the schedule identifier, the schedule start date time and end date time.
 * Note: This bean is created to ease the date time validation.
 * 
 */
public class SpecificDateScheduleDateTime implements Comparable<SpecificDateScheduleDateTime> {
	// An identifier for convenience (currently Schedule type(Specific date) and number, till we come up with some other mechanism to identify schedule) 
	// to know which schedule is being processed. First schedule specific date/recurring starts with scheduleIdentifier 0
	private String scheduleIdentifier;
	private Date startDateTime;
	private Date endDateTime;

	public SpecificDateScheduleDateTime(String scheduleIdentifier, Date startDateTime, Date endDateTime) {
		this.scheduleIdentifier = scheduleIdentifier;
		this.startDateTime = startDateTime;
		this.endDateTime = endDateTime;
	}

	public String getScheduleIdentifier() {
		return scheduleIdentifier;
	}

	public Date getStartDateTime() {
		return startDateTime;
	}

	public Date getEndDateTime() {
		return endDateTime;
	}

	@Override
	public int compareTo(SpecificDateScheduleDateTime compareSpecificDateScheduleDateTime) {
		if (compareSpecificDateScheduleDateTime == null)
			throw new NullPointerException("The SpecificDateScheduleDateTime object to be compared is null");

		Date thisDateTime = this.getStartDateTime();
		Date compareToDateTime = compareSpecificDateScheduleDateTime.getStartDateTime();

		if (thisDateTime == null || compareToDateTime == null)
			throw new NullPointerException("One of the date time value is null");

		return thisDateTime.compareTo(compareToDateTime);
	}

}
