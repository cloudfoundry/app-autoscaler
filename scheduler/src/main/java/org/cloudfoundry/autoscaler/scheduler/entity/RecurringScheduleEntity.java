package org.cloudfoundry.autoscaler.scheduler.entity;

import java.sql.Time;
import java.util.Arrays;
import java.util.Date;

import javax.persistence.Column;
import javax.persistence.Entity;
import javax.persistence.NamedQueries;
import javax.persistence.NamedQuery;
import javax.persistence.Table;
import javax.validation.constraints.NotNull;

import org.hibernate.annotations.Type;

@Entity
@Table(name = "app_scaling_recurring_schedule")
@NamedQueries({
		@NamedQuery(name = RecurringScheduleEntity.query_recurringSchedulesByAppId, query = RecurringScheduleEntity.jpql_recurringSchedulesByAppId) })
public class RecurringScheduleEntity extends ScheduleEntity {

	@Column(name = "start_time")
	@NotNull
	private Time startTime;

	@Column(name = "end_time")
	@NotNull
	private Time endTime;

	@Column(name = "start_date")
	private Date startDate;

	@Column(name = "end_date")
	private Date endDate;

	@Column(name = "day_of_week")
	@Type(type = "org.cloudfoundry.autoscaler.scheduler.entity.IntArrayUserType")
	private int[] dayOfWeek;

	@Column(name = "day_of_month")
	@Type(type = "org.cloudfoundry.autoscaler.scheduler.entity.IntArrayUserType")
	private int[] dayOfMonth;

	public int[] getDayOfWeek() {
		return dayOfWeek;
	}

	public void setDayOfWeek(int[] dayOfWeek) {
		this.dayOfWeek = dayOfWeek;
	}

	public int[] getDayOfMonth() {
		return dayOfMonth;
	}

	public void setDayOfMonth(int[] dayOfMonth) {
		this.dayOfMonth = dayOfMonth;
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

	public static final String query_recurringSchedulesByAppId = "RecurringScheduleEntity.schedulesByAppId";
	protected static final String jpql_recurringSchedulesByAppId = " FROM RecurringScheduleEntity" + " WHERE appId = :appId";

	@Override
	public String toString() {
		return "RecurringScheduleEntity [startTime=" + startTime + ", endTime=" + endTime + ", startDate=" + startDate
				+ ", endDate=" + endDate + ", dayOfWeek=" + Arrays.toString(dayOfWeek) + ", dayOfMonth="
				+ Arrays.toString(dayOfMonth) + "]";
	}

}
