package org.cloudfoundry.autoscaler.scheduler.entity;

import java.sql.Date;
import java.sql.Time;

import javax.persistence.Column;
import javax.persistence.Entity;
import javax.persistence.GeneratedValue;
import javax.persistence.GenerationType;
import javax.persistence.Id;
import javax.persistence.NamedQueries;
import javax.persistence.NamedQuery;
import javax.persistence.Table;

/**
 * @author Fujitsu
 *
 */
@Entity
@Table(name = "SCHEDULE")
@NamedQueries({
		@NamedQuery(name = ScheduleEntity.query_schedulesByAppId, query = ScheduleEntity.jpql_schedulesByAppId) })
public class ScheduleEntity {

	@Id
	@GeneratedValue(strategy = GenerationType.IDENTITY)
	@Column(name = "schedule_id")
	private Long scheduleId;

	@Column(name = "app_id", nullable = false)
	private String appId;

	@Column(name = "timezone")
	private String timezone;

	@Column(name = "start_date")
	private Date startDate;

	@Column(name = "end_date")
	private Date endDate;

	@Column(name = "start_time")
	private Time startTime;

	@Column(name = "end_time")
	private Time endTime;

	@Column(name = "instance_min_count")
	private int instanceMinCount;

	@Column(name = "instance_max_count")
	private int instanceMaxCount;

	@Column(name = "job_schedule_type")
	private String jobScheduleType;

	public Long getScheduleId() {
		return scheduleId;
	}

	public void setScheduleId(Long scheduleId) {
		this.scheduleId = scheduleId;
	}

	public String getAppId() {
		return appId;
	}

	public void setAppId(String appId) {
		this.appId = appId;
	}

	public String getTimezone() {
		return timezone;
	}

	public void setTimezone(String timezone) {
		this.timezone = timezone;
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

	public int getInstanceMinCount() {
		return instanceMinCount;
	}

	public void setInstanceMinCount(int instanceMinCount) {
		this.instanceMinCount = instanceMinCount;
	}

	public int getInstanceMaxCount() {
		return instanceMaxCount;
	}

	public void setInstanceMaxCount(int instanceMaxCount) {
		this.instanceMaxCount = instanceMaxCount;
	}

	public String getJobScheduleType() {
		return jobScheduleType;
	}

	public void setJobScheduleType(String jobScheduleType) {
		this.jobScheduleType = jobScheduleType;
	}

	public static final String query_schedulesByAppId = "ScheduleEntity.schedulesByAppId";
	protected static final String jpql_schedulesByAppId = " FROM ScheduleEntity" + " WHERE appId = :appId";

	@Override
	public String toString() {
		return "ScheduleEntity [scheduleId=" + scheduleId + ", appId=" + appId + ", timezone=" + timezone
				+ ", startDate=" + startDate + ", endDate=" + endDate + ", startTime=" + startTime + ", endTime="
				+ endTime + ", instanceMinCount=" + instanceMinCount + ", instanceMaxCount=" + instanceMaxCount
				+ ", jobScheduleType=" + jobScheduleType + "]";
	}

}
