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

	ScheduleBuilder(String timezone, int noOfSpecificDateSchedules, int noOfDOMRecurringSchedules, int noOfDOWRecurringSchedules) {
		this();

		schedules.setTimezone(timezone);
		schedules.setSpecific_date(new SpecificDateScheduleEntitiesBuilder(noOfSpecificDateSchedules).build());
		schedules.setRecurring_schedule(
				new RecurringScheduleEntitiesBuilder(noOfDOMRecurringSchedules, noOfDOWRecurringSchedules).build());

	}

	public ScheduleBuilder setTimezone(String timezone) {
		schedules.setTimezone(timezone);
		return this;
	}

	public ScheduleBuilder setSpecific_date(List<SpecificDateScheduleEntity> entities) {
		schedules.setSpecific_date(entities);
		return this;
	}

	public ScheduleBuilder setRecurring_schedule(List<RecurringScheduleEntity> entities) {
		schedules.setRecurring_schedule(entities);
		return this;
	}

	public Schedules build() {
		return schedules;
	}
}
