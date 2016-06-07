package org.cloudfoundry.autoscaler.scheduler.rest.model;

/**
 * @author Fujitsu
 *
 */
public class SpecificDateSchedule extends Schedule {
	private String start_date;
	private String end_date;

	public String getStart_date() {
		return start_date;
	}

	public void setStart_date(String start_date) {
		this.start_date = start_date;
	}

	public String getEnd_date() {
		return end_date;
	}

	public void setEnd_date(String end_date) {
		this.end_date = end_date;
	}

}
