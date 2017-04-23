package org.cloudfoundry.autoscaler.scheduler.entity;

import java.time.LocalDateTime;

import javax.persistence.Column;
import javax.persistence.GeneratedValue;
import javax.persistence.GenerationType;
import javax.persistence.Id;
import javax.persistence.MappedSuperclass;
import javax.persistence.SequenceGenerator;
import javax.validation.constraints.NotNull;

import org.cloudfoundry.autoscaler.scheduler.util.DateHelper;
import org.cloudfoundry.autoscaler.scheduler.util.DateTimeDeserializer;
import org.cloudfoundry.autoscaler.scheduler.util.DateTimeSerializer;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonProperty.Access;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;
import com.fasterxml.jackson.databind.annotation.JsonSerialize;

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
	
	@Column(name = "updated_at")
	@JsonProperty(value = "updated_at")
	@JsonFormat(pattern = DateHelper.DATE_TIME_FORMAT)
	@JsonDeserialize(using = DateTimeDeserializer.class)
	@JsonSerialize(using = DateTimeSerializer.class)
	private LocalDateTime updatedAt;
	

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
	
	public LocalDateTime getUpdatedAt() {
		return updatedAt;
	}

	public void setUpdatedAt(LocalDateTime updatedAt) {
		this.updatedAt = updatedAt;
	}

	@Override
	public boolean equals(Object o) {
		if (this == o)
			return true;
		if (o == null || getClass() != o.getClass())
			return false;

		ScheduleEntity that = (ScheduleEntity) o;

		if (id != null ? !id.equals(that.id) : that.id != null)
			return false;
		if (appId != null ? !appId.equals(that.appId) : that.appId != null)
			return false;
		if (timeZone != null ? !timeZone.equals(that.timeZone) : that.timeZone != null)
			return false;
		if (defaultInstanceMinCount != null ? !defaultInstanceMinCount.equals(that.defaultInstanceMinCount)
				: that.defaultInstanceMinCount != null)
			return false;
		if (defaultInstanceMaxCount != null ? !defaultInstanceMaxCount.equals(that.defaultInstanceMaxCount)
				: that.defaultInstanceMaxCount != null)
			return false;
		if (!instanceMinCount.equals(that.instanceMinCount))
			return false;
		if (!instanceMaxCount.equals(that.instanceMaxCount))
			return false;
		return initialMinInstanceCount != null ? initialMinInstanceCount.equals(that.initialMinInstanceCount)
				: that.initialMinInstanceCount == null;

	}

	@Override
	public int hashCode() {
		int result = id != null ? id.hashCode() : 0;
		result = 31 * result + (appId != null ? appId.hashCode() : 0);
		result = 31 * result + (timeZone != null ? timeZone.hashCode() : 0);
		result = 31 * result + (defaultInstanceMinCount != null ? defaultInstanceMinCount.hashCode() : 0);
		result = 31 * result + (defaultInstanceMaxCount != null ? defaultInstanceMaxCount.hashCode() : 0);
		result = 31 * result + instanceMinCount.hashCode();
		result = 31 * result + instanceMaxCount.hashCode();
		result = 31 * result + (initialMinInstanceCount != null ? initialMinInstanceCount.hashCode() : 0);
		return result;
	}

	@Override
	public String toString() {
		return "ScheduleEntity [id=" + id + ", appId=" + appId + ", timeZone=" + timeZone + ", defaultInstanceMinCount="
				+ defaultInstanceMinCount + ", defaultInstanceMaxCount=" + defaultInstanceMaxCount
				+ ", instanceMinCount=" + instanceMinCount + ", instanceMaxCount=" + instanceMaxCount
				+ ", initialMinInstanceCount=" + initialMinInstanceCount + "]";
	}

}
