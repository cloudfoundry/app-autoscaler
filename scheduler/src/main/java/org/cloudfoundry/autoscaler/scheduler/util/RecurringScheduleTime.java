package org.cloudfoundry.autoscaler.scheduler.util;

import java.util.Arrays;
import java.util.Date;
import java.util.List;
import java.util.stream.Collectors;

import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;

public class RecurringScheduleTime implements Comparable<RecurringScheduleTime> {
	private String scheduleIdentifier;
	private Date startDate;
	private Date endDate;
	private Date startTime;
	private Date endTime;

	private List<Integer> dayOfWeek = null;
	private List<Integer> dayOfMonth = null;

	public RecurringScheduleTime(String scheduleIdentifier, RecurringScheduleEntity recurringScheduleEntity) {
		this.scheduleIdentifier = scheduleIdentifier;
		this.startDate = recurringScheduleEntity.getStartDate();
		this.endDate = recurringScheduleEntity.getEndDate();
		this.startTime = recurringScheduleEntity.getStartTime();
		this.endTime = recurringScheduleEntity.getEndTime();

		if (recurringScheduleEntity.getDaysOfWeek() != null) {
			this.dayOfWeek = Arrays.stream(recurringScheduleEntity.getDaysOfWeek()).boxed().collect(Collectors.toList());
		}

		if (recurringScheduleEntity.getDaysOfMonth() != null) {
			this.dayOfMonth = Arrays.stream(recurringScheduleEntity.getDaysOfMonth()).boxed()
					.collect(Collectors.toList());
		}
	}

	String getScheduleIdentifier() {
		return scheduleIdentifier;
	}

	Date getStartTime() {
		return startTime;
	}

	Date getEndTime() {
		return endTime;
	}

	List<Integer> getDayOfWeek() {
		return this.dayOfWeek;
	}

	List<Integer> getDayOfMonth() {
		return this.dayOfMonth;
	}

	Date getStartDate() {
		return startDate;
	}

	Date getEndDate() {
		return endDate;
	}

	boolean hasDayOfWeek() {
		return getDayOfWeek() != null;
	}

	boolean hasDayOfMonth() {
		return getDayOfMonth() != null;
	}

	@Override
	public int compareTo(RecurringScheduleTime scheduleTime) {
		Date thisDateTime = this.getStartTime();
		Date compareToDateTime = scheduleTime.getStartTime();

		if (thisDateTime == null || compareToDateTime == null)
			throw new NullPointerException("One of the date time value is null");

		return thisDateTime.compareTo(compareToDateTime);
	}

}
