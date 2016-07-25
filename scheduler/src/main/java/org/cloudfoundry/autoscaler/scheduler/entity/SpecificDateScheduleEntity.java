package org.cloudfoundry.autoscaler.scheduler.entity;

import java.sql.Time;
import java.util.Date;

import javax.persistence.Column;
import javax.persistence.Entity;
import javax.persistence.NamedQueries;
import javax.persistence.NamedQuery;
import javax.persistence.Table;
import javax.validation.constraints.NotNull;

/**
 * 
 *
 */
@Entity
@Table(name = "app_scaling_specific_date_schedule")
@NamedQueries({
		@NamedQuery(name = SpecificDateScheduleEntity.query_specificDateSchedulesByAppId, query = SpecificDateScheduleEntity.jpql_specificDateSchedulesByAppId) })
public class SpecificDateScheduleEntity extends ScheduleEntity {

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

	public static final String query_specificDateSchedulesByAppId = "SpecificDateScheduleEntity.schedulesByAppId";
	protected static final String jpql_specificDateSchedulesByAppId = " FROM SpecificDateScheduleEntity" + " WHERE appId = :appId";

	@Override
	public String toString() {
		return "SpecificDateScheduleEntity [startDate=" + startDate + ", endDate=" + endDate + ", startTime="
				+ startTime + ", endTime=" + endTime + "]";
	}

}
