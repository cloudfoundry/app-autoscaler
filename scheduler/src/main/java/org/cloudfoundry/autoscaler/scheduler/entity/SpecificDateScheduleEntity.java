package org.cloudfoundry.autoscaler.scheduler.entity;

import java.util.Date;

import javax.persistence.Entity;
import javax.persistence.NamedQueries;
import javax.persistence.NamedQuery;
import javax.persistence.Table;
import javax.validation.constraints.NotNull;

import org.cloudfoundry.autoscaler.scheduler.util.DateHelper;
import org.cloudfoundry.autoscaler.scheduler.util.DateTimeSerializer;
import org.cloudfoundry.autoscaler.scheduler.util.DateTimeDeserializer;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;
import com.fasterxml.jackson.databind.annotation.JsonSerialize;

/**
 * 
 *
 */
@Entity
@Table(name = "app_scaling_specific_date_schedule")
@NamedQueries({
		@NamedQuery(name = SpecificDateScheduleEntity.query_specificDateSchedulesByAppId, query = SpecificDateScheduleEntity.jpql_specificDateSchedulesByAppId) })
public class SpecificDateScheduleEntity extends ScheduleEntity {
	
	@JsonFormat(pattern = DateHelper.DATE_TIME_FORMAT)
	@JsonDeserialize(using = DateTimeDeserializer.class)
	@JsonSerialize(using = DateTimeSerializer.class)
	@NotNull
	private Date start_date_time;
	
	@JsonFormat(pattern = DateHelper.DATE_TIME_FORMAT)
	@JsonDeserialize(using = DateTimeDeserializer.class)
	@JsonSerialize(using = DateTimeSerializer.class)
	@NotNull
	private Date end_date_time;

	public Date getStart_date_time() {
		return start_date_time;
	}

	public void setStart_date_time(Date start_date_time) {
		this.start_date_time = start_date_time;
	}

	public Date getEnd_date_time() {
		return end_date_time;
	}

	public void setEnd_date_time(Date end_date_time) {
		this.end_date_time = end_date_time;
	}

	public static final String query_specificDateSchedulesByAppId = "SpecificDateScheduleEntity.schedulesByAppId";
	protected static final String jpql_specificDateSchedulesByAppId = " FROM SpecificDateScheduleEntity"
			+ " WHERE app_id = :app_id";

	@Override
	public String toString() {
		return "SpecificDateScheduleEntity [start_date_time=" + start_date_time + ", end_date_time=" + end_date_time
				+ "]";
	}

}
