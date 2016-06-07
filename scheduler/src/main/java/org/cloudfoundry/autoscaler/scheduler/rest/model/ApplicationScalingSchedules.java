package org.cloudfoundry.autoscaler.scheduler.rest.model;

import java.util.List;

/**
 * @author Fujitsu
 *
 */
public class ApplicationScalingSchedules {
	private String app_id;
	private String timezone;
	private List<SpecificDateSchedule> specific_date;

	public String getApp_id() {
		return app_id;
	}

	public void setApp_id(String app_id) {
		this.app_id = app_id;
	}

	public String getTimezone() {
		return timezone;
	}

	public void setTimezone(String timezone) {
		this.timezone = timezone;
	}

	public List<SpecificDateSchedule> getSpecific_date() {
		return specific_date;
	}

	public void setSpecific_date(List<SpecificDateSchedule> specific_date) {
		this.specific_date = specific_date;
	}

}