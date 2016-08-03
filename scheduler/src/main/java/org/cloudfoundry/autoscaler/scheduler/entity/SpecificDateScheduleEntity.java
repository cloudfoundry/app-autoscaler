package org.cloudfoundry.autoscaler.scheduler.entity;

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

	@Column(name = "start_date_time")
	@NotNull
	private Date startDateTime;

	@Column(name = "end_date_time")
	@NotNull
	private Date endDateTime;
	
	public Date getStartDateTime() {
		return startDateTime;
	}

	public void setStartDateTime(Date startDateTime) {
		this.startDateTime = startDateTime;
	}

	public Date getEndDateTime() {
		return endDateTime;
	}

	public void setEndDateTime(Date endDateTime) {
		this.endDateTime = endDateTime;
	}

	public static final String query_specificDateSchedulesByAppId = "SpecificDateScheduleEntity.schedulesByAppId";
	protected static final String jpql_specificDateSchedulesByAppId = " FROM SpecificDateScheduleEntity" + " WHERE appId = :appId";

	@Override
	public String toString() {
		return "SpecificDateScheduleEntity [startDateTime=" + startDateTime + ", endDateTime=" + endDateTime + "]";
	}

}
