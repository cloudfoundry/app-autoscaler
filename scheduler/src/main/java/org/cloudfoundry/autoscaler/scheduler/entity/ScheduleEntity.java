package org.cloudfoundry.autoscaler.scheduler.entity;

import javax.persistence.Column;
import javax.persistence.GeneratedValue;
import javax.persistence.GenerationType;
import javax.persistence.Id;
import javax.persistence.MappedSuperclass;
import javax.validation.constraints.NotNull;

/**
 * 
 *
 */

@MappedSuperclass
public class ScheduleEntity {

	@Id
	@GeneratedValue(strategy = GenerationType.IDENTITY)
	@Column(name = "schedule_id")
	private Long id;

	@NotNull
	private String app_id;

	@NotNull
	private String timezone;

	@NotNull
	private Integer default_instance_min_count;

	@NotNull
	private Integer default_instance_max_count;

	@NotNull
	private Integer instance_min_count;

	@NotNull
	private Integer instance_max_count;

	private Integer initial_min_instance_count;

	public Long getId() {
		return id;
	}

	public void setId(Long id) {
		this.id = id;
	}

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

	public Integer getDefault_instance_min_count() {
		return default_instance_min_count;
	}

	public void setDefault_instance_min_count(Integer default_instance_min_count) {
		this.default_instance_min_count = default_instance_min_count;
	}

	public Integer getDefault_instance_max_count() {
		return default_instance_max_count;
	}

	public void setDefault_instance_max_count(Integer default_instance_max_count) {
		this.default_instance_max_count = default_instance_max_count;
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

	public Integer getInitial_min_instance_count() {
		return initial_min_instance_count;
	}

	public void setInitial_min_instance_count(Integer initial_min_instance_count) {
		this.initial_min_instance_count = initial_min_instance_count;
	}

	@Override
	public String toString() {
		return "ScheduleEntity [id=" + id + ", app_id=" + app_id + ", timezone=" + timezone
				+ ", default_instance_min_count=" + default_instance_min_count + ", default_instance_max_count="
				+ default_instance_max_count + ", instance_min_count=" + instance_min_count + ", instance_max_count="
				+ instance_max_count + ", initial_min_instance_count=" + initial_min_instance_count + "]";
	}

}
