package org.cloudfoundry.autoscaler.scheduler.entity;

import java.sql.Time;
import java.util.Arrays;
import java.util.Date;

import javax.persistence.Entity;
import javax.persistence.NamedQueries;
import javax.persistence.NamedQuery;
import javax.persistence.Table;
import javax.validation.constraints.NotNull;

import org.cloudfoundry.autoscaler.scheduler.util.DateDeserializer;
import org.cloudfoundry.autoscaler.scheduler.util.DateHelper;
import org.cloudfoundry.autoscaler.scheduler.util.DateSerializer;
import org.cloudfoundry.autoscaler.scheduler.util.SqlTimeDeserializer;
import org.cloudfoundry.autoscaler.scheduler.util.SqlTimeSerializer;
import org.hibernate.annotations.Type;
import org.springframework.format.annotation.DateTimeFormat;

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
	private Time start_time;

	@JsonDeserialize(using = SqlTimeDeserializer.class)
	@JsonSerialize(using = SqlTimeSerializer.class)
	@NotNull
	private Time end_time;

	@JsonDeserialize(using = DateDeserializer.class)
	@JsonSerialize(using = DateSerializer.class)
	private Date start_date;

	@JsonDeserialize(using = DateDeserializer.class)
	@JsonSerialize(using = DateSerializer.class)
	private Date end_date;

	@Type(type = "org.cloudfoundry.autoscaler.scheduler.entity.BitsetUserType")
	private int[] days_of_week;

	@Type(type = "org.cloudfoundry.autoscaler.scheduler.entity.BitsetUserType")
	private int[] days_of_month;

	public int[] getDays_of_week() {
		return days_of_week;
	}

	public void setDays_of_week(int[] days_of_week) {
		this.days_of_week = days_of_week;
	}

	public int[] getDays_of_month() {
		return days_of_month;
	}

	public void setDay_of_month(int[] days_of_month) {
		this.days_of_month = days_of_month;
	}

	public Time getStart_time() {
		return start_time;
	}

	public void setStart_time(Time start_time) {
		this.start_time = start_time;
	}

	public Time getEnd_time() {
		return end_time;
	}

	public void setEnd_time(Time end_time) {
		this.end_time = end_time;
	}

	public Date getStart_date() {
		return start_date;
	}

	public void setStart_date(Date start_date) {
		this.start_date = start_date;
	}

	public Date getEnd_date() {
		return end_date;
	}

	public void setEnd_date(Date end_date) {
		this.end_date = end_date;
	}

	public static final String query_recurringSchedulesByAppId = "RecurringScheduleEntity.schedulesByAppId";
	protected static final String jpql_recurringSchedulesByAppId = " FROM RecurringScheduleEntity"
			+ " WHERE app_id = :app_id";

	@Override
	public String toString() {
		return "RecurringScheduleEntity [startTime=" + start_time + ", endTime=" + end_time + ", startDate="
				+ start_date + ", endDate=" + end_date + ", dayOfWeek=" + Arrays.toString(days_of_week) + ", dayOfMonth="
				+ Arrays.toString(days_of_month) + "]";
	}

}
