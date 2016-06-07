package org.cloudfoundry.autoscaler.scheduler.rest.model;

/**
 * @author Fujitsu
 *
 */
public class Schedule {
	private String start_time;
	private String end_time;
	private int instance_min_count;
	private int instance_max_count;

	public String getStart_time() {
		return start_time;
	}

	public void setStart_time(String start_time) {
		this.start_time = start_time;
	}

	public String getEnd_time() {
		return end_time;
	}

	public void setEnd_time(String end_time) {
		this.end_time = end_time;
	}

	public int getInstance_min_count() {
		return instance_min_count;
	}

	public void setInstance_min_count(int instance_min_count) {
		this.instance_min_count = instance_min_count;
	}

	public int getInstance_max_count() {
		return instance_max_count;
	}

	public void setInstance_max_count(int instance_max_count) {
		this.instance_max_count = instance_max_count;
	}

}