package org.cloudfoundry.autoscaler.scheduler.rest.model;

import java.util.List;

import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;

/**
 * 
 *
 */
public class Schedules {
	String timezone;

	private List<SpecificDateScheduleEntity> specific_date;
	private List<RecurringScheduleEntity> recurring_schedule;

	public boolean hasSchedules() {
		if ((specific_date == null || specific_date.isEmpty())
				&& (recurring_schedule == null || recurring_schedule.isEmpty())) {
			return false;
		}
		return true;
	}

	public String getTimezone() {
		return timezone;
	}

	public void setTimezone(String timezone) {
		this.timezone = timezone;
	}

	public List<SpecificDateScheduleEntity> getSpecific_date() {
		return specific_date;
	}

	public void setSpecific_date(List<SpecificDateScheduleEntity> specific_date) {
		this.specific_date = specific_date;
	}

	public List<RecurringScheduleEntity> getRecurring_schedule() {
		return recurring_schedule;
	}

	public void setRecurring_schedule(List<RecurringScheduleEntity> recurring_schedule) {
		this.recurring_schedule = recurring_schedule;
	}

}