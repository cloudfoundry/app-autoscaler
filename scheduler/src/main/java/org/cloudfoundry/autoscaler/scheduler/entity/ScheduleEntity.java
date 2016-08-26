package org.cloudfoundry.autoscaler.scheduler.entity;

import javax.persistence.Column;
import javax.persistence.GeneratedValue;
import javax.persistence.GenerationType;
import javax.persistence.Id;
import javax.persistence.MappedSuperclass;
import javax.validation.constraints.NotNull;

import com.fasterxml.jackson.annotation.JsonProperty;

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
	@JsonProperty(value = "app_id")
	@Column(name = "app_id")
	private String appId;

	@NotNull
	@JsonProperty(value = "timezone")
	@Column(name = "timezone")
	private String timeZone;

	@NotNull
	@Column(name = "default_instance_min_count")
	@JsonProperty(value = "default_instance_min_count")
	private Integer defaultInstanceMinCount;

	@NotNull
	@Column(name = "default_instance_max_count")
	@JsonProperty(value = "default_instance_max_count")
	private Integer defaultInstanceMaxCount;

	@NotNull
	@Column(name = "instance_min_count")
	@JsonProperty(value = "instance_min_count")
	private Integer instanceMinCount;

	@NotNull
	@Column(name = "instance_max_count")
	@JsonProperty(value = "instance_max_count")
	private Integer instanceMaxCount;

	@Column(name = "initial_min_instance_count")
	@JsonProperty(value = "initial_min_instance_count")
	private Integer initialMinInstanceCount;

	public Long getId() {
		return id;
	}

	public void setId(Long id) {
		this.id = id;
	}

	public String getAppId() {
		return appId;
	}

	public void setAppId(String appId) {
		this.appId = appId;
	}

	public String getTimeZone() {
		return timeZone;
	}

	public void setTimeZone(String timeZone) {
		this.timeZone = timeZone;
	}

	public Integer getDefaultInstanceMinCount() {
		return defaultInstanceMinCount;
	}

	public void setDefaultInstanceMinCount(Integer defaultInstanceMinCount) {
		this.defaultInstanceMinCount = defaultInstanceMinCount;
	}

	public Integer getDefaultInstanceMaxCount() {
		return defaultInstanceMaxCount;
	}

	public void setDefaultInstanceMaxCount(Integer defaultInstanceMaxCount) {
		this.defaultInstanceMaxCount = defaultInstanceMaxCount;
	}

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

	public Integer getInitialMinInstanceCount() {
		return initialMinInstanceCount;
	}

	public void setInitialMinInstanceCount(Integer initialMinInstanceCount) {
		this.initialMinInstanceCount = initialMinInstanceCount;
	}

	@Override
	public String toString() {
		return "ScheduleEntity [id=" + id + ", appId=" + appId + ", timeZone=" + timeZone + ", defaultInstanceMinCount="
				+ defaultInstanceMinCount + ", defaultInstanceMaxCount=" + defaultInstanceMaxCount
				+ ", instanceMinCount=" + instanceMinCount + ", instanceMaxCount=" + instanceMaxCount
				+ ", initialMinInstanceCount=" + initialMinInstanceCount + "]";
	}

}
