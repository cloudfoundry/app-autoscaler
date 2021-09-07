package org.cloudfoundry.autoscaler.scheduler.util;

import java.time.LocalDate;
import java.time.LocalTime;
import java.util.Arrays;
import java.util.List;
import java.util.stream.Collectors;
import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;

public class RecurringScheduleTime implements Comparable<RecurringScheduleTime> {
  private String scheduleIdentifier;
  private LocalDate startDate;
  private LocalDate endDate;
  private LocalTime startTime;
  private LocalTime endTime;

  private List<Integer> dayOfWeek = null;
  private List<Integer> dayOfMonth = null;

  public RecurringScheduleTime(
      String scheduleIdentifier, RecurringScheduleEntity recurringScheduleEntity) {
    this.scheduleIdentifier = scheduleIdentifier;
    this.startDate = recurringScheduleEntity.getStartDate();
    this.endDate = recurringScheduleEntity.getEndDate();
    this.startTime = recurringScheduleEntity.getStartTime();
    this.endTime = recurringScheduleEntity.getEndTime();

    if (recurringScheduleEntity.getDaysOfWeek() != null) {
      this.dayOfWeek =
          Arrays.stream(recurringScheduleEntity.getDaysOfWeek())
              .boxed()
              .collect(Collectors.toList());
    }

    if (recurringScheduleEntity.getDaysOfMonth() != null) {
      this.dayOfMonth =
          Arrays.stream(recurringScheduleEntity.getDaysOfMonth())
              .boxed()
              .collect(Collectors.toList());
    }
  }

  String getScheduleIdentifier() {
    return scheduleIdentifier;
  }

  LocalTime getStartTime() {
    return startTime;
  }

  LocalTime getEndTime() {
    return endTime;
  }

  List<Integer> getDayOfWeek() {
    return this.dayOfWeek;
  }

  List<Integer> getDayOfMonth() {
    return this.dayOfMonth;
  }

  LocalDate getStartDate() {
    return startDate;
  }

  LocalDate getEndDate() {
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
    LocalTime thisDateTime = this.getStartTime();
    LocalTime compareToDateTime = scheduleTime.getStartTime();

    if (thisDateTime != null && compareToDateTime != null) {
      return thisDateTime.compareTo(compareToDateTime);
    }
    throw new NullPointerException("One of the date time value is null");
  }
}
