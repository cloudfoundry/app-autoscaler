package org.cloudfoundry.autoscaler.scheduler.entity;

import javax.persistence.Column;
import javax.persistence.GeneratedValue;
import javax.persistence.GenerationType;
import javax.persistence.Id;
import javax.persistence.MappedSuperclass;
import javax.persistence.SequenceGenerator;
import javax.validation.constraints.NotNull;

import com.fasterxml.jackson.annotation.JsonProperty;

import io.swagger.annotations.ApiModel;
import io.swagger.annotations.ApiModelProperty;

@ApiModel
@MappedSuperclass
public class ScheduleEntity {

	@ApiModelProperty(hidden = true)
	@Id
	@GeneratedValue(strategy = GenerationType.SEQUENCE, generator = "schedule_id_generator")
	@SequenceGenerator(name = "schedule_id_generator", sequenceName = "schedule_id_sequence", allocationSize = 1)
	@Column(name = "schedule_id")
	private Long id;

	@ApiModelProperty(hidden = true)
	@NotNull
	@JsonProperty(value = "app_id")
	@Column(name = "app_id")
	private String appId;

	@ApiModelProperty(hidden = true)
	@NotNull
	@JsonProperty(value = "timezone")
	@Column(name = "timezone")
	private String timeZone;

	@ApiModelProperty(hidden = true)
	@NotNull
	@Column(name = "default_instance_min_count")
	@JsonProperty(value = "default_instance_min_count")
	private Integer defaultInstanceMinCount;

	@ApiModelProperty(hidden = true)
	@NotNull
	@Column(name = "default_instance_max_count")
	@JsonProperty(value = "default_instance_max_count")
	private Integer defaultInstanceMaxCount;

	@ApiModelProperty(required = true, position = 10)
	@NotNull
	@Column(name = "instance_min_count")
	@JsonProperty(value = "instance_min_count")
	private Integer instanceMinCount;

	@ApiModelProperty(required = true, position = 11)
	@NotNull
	@Column(name = "instance_max_count")
	@JsonProperty(value = "instance_max_count")
	private Integer instanceMaxCount;

	@ApiModelProperty(required = true, position = 12)
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
