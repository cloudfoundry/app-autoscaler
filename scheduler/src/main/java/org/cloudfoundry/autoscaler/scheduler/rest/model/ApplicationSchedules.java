package org.cloudfoundry.autoscaler.scheduler.rest.model;

import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * 
 *
 */
public class ApplicationSchedules {
	@JsonProperty(value = "instance_min_count")
	Integer instanceMinCount;

	@JsonProperty(value = "instance_max_count")
	Integer instanceMaxCount;
	Schedules schedules;

	public Integer getInstanceMinCount() {
		return instanceMinCount;
	}

	public void setInstanceMinCount(Integer instanceMinCount) {
		this.instanceMinCount = instanceMinCount;
	}

	public Integer getInstanceMaxCount() {
		return instanceMaxCount;
	}

	public void setInstanceMaxCount(Integer instanceMaxCount) {
		this.instanceMaxCount = instanceMaxCount;
	}

	public Schedules getSchedules() {
		return schedules;
	}

	public void setSchedules(Schedules schedules) {
		this.schedules = schedules;
	}

	@Override
	public String toString() {
		return "ApplicationPolicy [instanceMinCount=" + instanceMinCount + ", instanceMaxCount=" + instanceMaxCount
				+ ", schedules=" + schedules + "]";
	}

}