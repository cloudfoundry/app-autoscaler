package org.cloudfoundry.autoscaler.scheduler.rest.model;

import java.util.List;

import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;

import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * 
 *
 */
public class Schedules {
	@JsonProperty(value = "timezone")
	String timeZone;

	@JsonProperty(value = "specific_date")
	private List<SpecificDateScheduleEntity> specificDate;
	@JsonProperty(value = "recurring_schedule")
	private List<RecurringScheduleEntity> recurringSchedule;

	public boolean hasSchedules() {
		if ((specificDate == null || specificDate.isEmpty())
				&& (recurringSchedule == null || recurringSchedule.isEmpty())) {
			return false;
		}
		return true;
	}

	public String getTimeZone() {
		return timeZone;
	}

	public void setTimeZone(String timeZone) {
		this.timeZone = timeZone;
	}

	public List<SpecificDateScheduleEntity> getSpecificDate() {
		return specificDate;
	}

	public void setSpecificDate(List<SpecificDateScheduleEntity> specificDate) {
		this.specificDate = specificDate;
	}

	public List<RecurringScheduleEntity> getRecurringSchedule() {
		return recurringSchedule;
	}

	public void setRecurringSchedule(List<RecurringScheduleEntity> recurringSchedule) {
		this.recurringSchedule = recurringSchedule;
	}

	@Override
	public String toString() {
		return "Schedules [timeZone=" + timeZone + ", specificDate=" + specificDate + ", recurringSchedule="
				+ recurringSchedule + "]";
	}

}