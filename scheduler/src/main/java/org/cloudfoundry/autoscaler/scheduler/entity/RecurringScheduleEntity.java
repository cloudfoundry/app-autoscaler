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

import org.cloudfoundry.autoscaler.scheduler.util.DateDeserializer;
import org.cloudfoundry.autoscaler.scheduler.util.DateSerializer;
import org.cloudfoundry.autoscaler.scheduler.util.SqlTimeDeserializer;
import org.cloudfoundry.autoscaler.scheduler.util.SqlTimeSerializer;
import org.hibernate.annotations.Type;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;
import com.fasterxml.jackson.databind.annotation.JsonSerialize;

@Entity
@Table(name = "app_scaling_recurring_schedule")
@NamedQueries({
		@NamedQuery(name = RecurringScheduleEntity.query_recurringSchedulesByAppId, query = RecurringScheduleEntity.jpql_recurringSchedulesByAppId) })
public class RecurringScheduleEntity extends ScheduleEntity {

	@JsonDeserialize(using = SqlTimeDeserializer.class)
	@JsonSerialize(using = SqlTimeSerializer.class)
	@NotNull
	@Column(name = "start_time")
	@JsonProperty(value = "start_time")
	private Time startTime;

	@JsonDeserialize(using = SqlTimeDeserializer.class)
	@JsonSerialize(using = SqlTimeSerializer.class)
	@NotNull
	@Column(name = "end_time")
	@JsonProperty(value = "end_time")
	private Time endTime;

	@JsonDeserialize(using = DateDeserializer.class)
	@JsonSerialize(using = DateSerializer.class)
	@Column(name = "start_date")
	@JsonProperty(value = "start_date")
	private Date startDate;

	@JsonDeserialize(using = DateDeserializer.class)
	@JsonSerialize(using = DateSerializer.class)
	@Column(name = "end_date")
	@JsonProperty(value = "end_date")
	private Date endDate;

	@Type(type = "org.cloudfoundry.autoscaler.scheduler.entity.BitsetUserType")
	@Column(name = "days_of_week")
	@JsonProperty(value = "days_of_week")
	private int[] daysOfWeek;

	@Type(type = "org.cloudfoundry.autoscaler.scheduler.entity.BitsetUserType")
	@Column(name = "days_of_month")
	@JsonProperty(value = "days_of_month")
	private int[] daysOfMonth;

	public int[] getDaysOfWeek() {
		return daysOfWeek;
	}

	public void setDaysOfWeek(int[] daysOfWeek) {
		this.daysOfWeek = daysOfWeek;
	}

	public int[] getDaysOfMonth() {
		return daysOfMonth;
	}

	public void setDayOfMonth(int[] daysOfMonth) {
		this.daysOfMonth = daysOfMonth;
	}

	@JsonProperty("start_time")
	public Time getStartTime() {
		return startTime;
	}

	@JsonProperty("start_time")
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
	protected static final String jpql_recurringSchedulesByAppId = " FROM RecurringScheduleEntity"
			+ " WHERE app_id = :appId";

	@Override
	public String toString() {
		return "RecurringScheduleEntity [startTime=" + startTime + ", endTime=" + endTime + ", startDate=" + startDate
				+ ", endDate=" + endDate + ", dayOfWeek=" + Arrays.toString(daysOfWeek) + ", dayOfMonth="
				+ Arrays.toString(daysOfMonth) + "]";
	}

}
