package org.cloudfoundry.autoscaler.scheduler.util;

import java.util.Arrays;
import java.util.Date;
import java.util.List;

import org.apache.commons.lang3.ArrayUtils;
import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;

public class RecurringScheduleTime implements Comparable<RecurringScheduleTime> {
	private String scheduleIdentifier;
	private Date startDate;
	private Date endDate;
	private Date startTime;
	private Date endTime;

	private Integer[] dayOfWeek;
	private Integer[] dayOfMonth;

	public RecurringScheduleTime(String scheduleIdentifier, RecurringScheduleEntity recurringScheduleEntity) {
		this.scheduleIdentifier = scheduleIdentifier;
		this.startDate = recurringScheduleEntity.getStartDate();
		this.endDate = recurringScheduleEntity.getEndDate();
		this.startTime = recurringScheduleEntity.getStartTime();
		this.endTime = recurringScheduleEntity.getEndTime();
		this.dayOfWeek = ArrayUtils.toObject(recurringScheduleEntity.getDayOfWeek());
		this.dayOfMonth = ArrayUtils.toObject(recurringScheduleEntity.getDayOfMonth());
	}

	public String getScheduleIdentifier() {
		return scheduleIdentifier;
	}

	public Date getStartTime() {
		return startTime;
	}

	public Date getEndTime() {
		return endTime;
	}

	public List<Integer> getDayOfWeek() {
		if (this.dayOfWeek != null) {
			return Arrays.asList(this.dayOfWeek);
		} else {
			return null;
		}
	}

	public List<Integer> getDayOfMonth() {
		if (this.dayOfMonth != null) {
			return Arrays.asList(this.dayOfMonth);
		} else {
			return null;
		}
	}

	public Date getStartDate() {
		return startDate;
	}

	public Date getEndDate() {
		return endDate;
	}

	public boolean hasDayOfWeek() {
		return getDayOfWeek() != null;
	}

	public boolean hasDayOfMonth() {
		return getDayOfMonth() != null;
	}

	@Override
	public int compareTo(RecurringScheduleTime scheduleTime) {
		if (scheduleTime == null)
			throw new NullPointerException("The RecurringScheduleTime object to be compared is null");

		Date thisDateTime = this.getStartTime();
		Date compareToDateTime = scheduleTime.getStartTime();

		if (thisDateTime == null || compareToDateTime == null)
			throw new NullPointerException("One of the date time value is null");

		return thisDateTime.compareTo(compareToDateTime);
	}

}
