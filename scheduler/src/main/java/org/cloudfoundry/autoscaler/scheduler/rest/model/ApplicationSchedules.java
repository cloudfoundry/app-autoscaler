package org.cloudfoundry.autoscaler.scheduler.rest.model;

/**
 * 
 *
 */
public class ApplicationSchedules {
	Integer instance_min_count;
	Integer instance_max_count;
	Schedules schedules;

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

	public Schedules getSchedules() {
		return schedules;
	}

	public void setSchedules(Schedules schedules) {
		this.schedules = schedules;
	}

	@Override
	public String toString() {
		return "ApplicationPolicy [instance_min_count=" + instance_min_count + ", instance_max_count="
				+ instance_max_count + ", schedules=" + schedules + "]";
	}

}