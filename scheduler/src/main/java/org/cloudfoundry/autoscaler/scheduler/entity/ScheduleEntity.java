package org.cloudfoundry.autoscaler.scheduler.entity;

import java.sql.Time;
import java.util.Date;

import javax.persistence.Column;
import javax.persistence.Entity;
import javax.persistence.GeneratedValue;
import javax.persistence.GenerationType;
import javax.persistence.Id;
import javax.persistence.NamedQueries;
import javax.persistence.NamedQuery;
import javax.persistence.Table;
import javax.validation.constraints.NotNull;

/**
 * 
 *
 */
@Entity
@Table(name = "app_scaling_schedule")
@NamedQueries({
		@NamedQuery(name = ScheduleEntity.query_schedulesByAppId, query = ScheduleEntity.jpql_schedulesByAppId) })
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

	@Column(name = "start_date")
	@NotNull
	private Date startDate;

	@Column(name = "end_date")
	@NotNull
	private Date endDate;

	@Column(name = "start_time")
	private Time startTime;

	@Column(name = "end_time")
	@NotNull
	private Time endTime;

	@Column(name = "instance_min_count")
	@NotNull
	private Integer instanceMinCount;

	@Column(name = "instance_max_count")
	@NotNull
	private Integer instanceMaxCount;

	@Column(name = "schedule_type")
	@NotNull
	private String scheduleType;

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

	public void setTimeZone(String timezone) {
		this.timeZone = timezone;
	}

	public Date getStartDate() {
		return startDate;
	}

	public void setStartDate(Date startDate) {
		this.startDate = startDate;
	}

	public Date getEndDate() {
		return endDate;
	}

	public void setEndDate(Date endDate) {
		this.endDate = endDate;
	}

	public Time getStartTime() {
		return startTime;
	}

	public void setStartTime(Time startTime) {
		this.startTime = startTime;
	}

	public Time getEndTime() {
		return endTime;
	}

	public void setEndTime(Time endTime) {
		this.endTime = endTime;
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

	public String getScheduleType() {
		return scheduleType;
	}

	public void setScheduleType(String scheduleType) {
		this.scheduleType = scheduleType;
	}



	public static final String query_schedulesByAppId = "ScheduleEntity.schedulesByAppId";
	protected static final String jpql_schedulesByAppId = " FROM ScheduleEntity" + " WHERE appId = :appId";

	@Override
	public String toString() {
		return "ScheduleEntity [id=" + id + ", appId=" + appId + ", timeZone=" + timeZone + ", defaultInstanceMinCount="
				+ defaultInstanceMinCount + ", defaultInstanceMaxCount=" + defaultInstanceMaxCount + ", startDate="
				+ startDate + ", endDate=" + endDate + ", startTime=" + startTime + ", endTime=" + endTime
				+ ", instanceMinCount=" + instanceMinCount + ", instanceMaxCount=" + instanceMaxCount
				+ ", scheduleType=" + scheduleType + "]";
	}

}
