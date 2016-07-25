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

	@Column(name = "app_id")
	@NotNull
	private String appId;

	@Column(name = "timezone")
	@NotNull
	private String timeZone;
	
	@Column(name = "default_instance_min_count")
	@NotNull
	private Integer defaultInstanceMinCount;
	
	@Column(name = "default_instance_max_count")
	@NotNull
	private Integer defaultInstanceMaxCount;

	@Column(name = "instance_min_count")
	@NotNull
	private Integer instanceMinCount;

	@Column(name = "instance_max_count")
	@NotNull
	private Integer instanceMaxCount;

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

	@Override
	public String toString() {
		return "ScheduleEntity [id=" + id + ", appId=" + appId + ", timeZone=" + timeZone + ", defaultInstanceMinCount="
				+ defaultInstanceMinCount + ", defaultInstanceMaxCount=" + defaultInstanceMaxCount
				+ ", instanceMinCount=" + instanceMinCount + ", instanceMaxCount=" + instanceMaxCount + "]";
	}

}
