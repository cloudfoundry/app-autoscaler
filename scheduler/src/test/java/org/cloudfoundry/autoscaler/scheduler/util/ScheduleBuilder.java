package org.cloudfoundry.autoscaler.scheduler.util;

import java.util.List;
import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.Schedules;

public class ScheduleBuilder {
  Schedules schedules;

  public ScheduleBuilder() {
    this.schedules = new Schedules();
  }

  ScheduleBuilder(
      String timeZone,
      int noOfSpecificDateSchedules,
      int noOfDomRecurringSchedules,
      int noOfDowRecurringSchedules) {
    this();

    schedules.setTimeZone(timeZone);
    schedules.setSpecificDate(
        new SpecificDateScheduleEntitiesBuilder(noOfSpecificDateSchedules).build());
    schedules.setRecurringSchedule(
        new RecurringScheduleEntitiesBuilder(noOfDomRecurringSchedules, noOfDowRecurringSchedules)
            .build());
  }

  public ScheduleBuilder setTimeZone(String timeZone) {
    schedules.setTimeZone(timeZone);
    return this;
  }

  public ScheduleBuilder setSpecificDate(List<SpecificDateScheduleEntity> entities) {
    schedules.setSpecificDate(entities);
    return this;
  }

  public ScheduleBuilder setRecurringSchedule(List<RecurringScheduleEntity> entities) {
    schedules.setRecurringSchedule(entities);
    return this;
  }

  public Schedules build() {
    return schedules;
  }
}
