package org.cloudfoundry.autoscaler.scheduler.rest.model;

import java.util.List;

import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;

/**
 * 
 *
 */
public class ApplicationScalingSchedules {
	String timeZone;
	Integer instance_min_count;
	Integer instance_max_count;
	private List<SpecificDateScheduleEntity> specific_date;
	private List<ScheduleEntity> recurring_schedule;

	public boolean hasSchedules() {
		if ((specific_date == null || specific_date.isEmpty())
				&& (recurring_schedule == null || recurring_schedule.isEmpty())) {
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

	public Integer getInstance_min_count() {
		return instance_min_count;
	}

	public void setInstance_min_count(Integer instance_min_count) {
		this.instance_min_count = instance_min_count;
	}

	public Integer getInstance_max_count() {
		return instance_max_count;
	}

	public void setInstance_max_count(Integer instance_max_count) {
		this.instance_max_count = instance_max_count;
	}

	public List<SpecificDateScheduleEntity> getSpecific_date() {
		return specific_date;
	}

	public void setSpecific_date(List<SpecificDateScheduleEntity> specific_date) {
		this.specific_date = specific_date;
	}

	public List<ScheduleEntity> getRecurring_schedule() {
		return recurring_schedule;
	}

	public void setRecurring_schedule(List<ScheduleEntity> recurring_schedule) {
		this.recurring_schedule = recurring_schedule;
	}

}