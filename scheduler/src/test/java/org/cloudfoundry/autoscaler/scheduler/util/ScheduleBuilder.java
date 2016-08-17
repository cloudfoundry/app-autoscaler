package org.cloudfoundry.autoscaler.scheduler.util;

import java.util.List;

import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationScalingSchedules;

public class ScheduleBuilder {
	ApplicationScalingSchedules schedules;

	public ScheduleBuilder() {
		this.schedules = new ApplicationScalingSchedules();
	}

	ScheduleBuilder(int instanceMaxCount, int instanceMinCount, String timeZone, int noOfSpecificDateSchedules,
			int noOfDOMRecurringSchedules, int noOfDOWRecurringSchedules) {
		this();

		schedules.setTimeZone(timeZone);
		schedules.setInstance_max_count(instanceMaxCount);
		schedules.setInstance_min_count(instanceMinCount);
		schedules.setSpecific_date(new SpecificDateScheduleEntitiesBuilder(noOfSpecificDateSchedules).build());
		schedules.setRecurring_schedule(
				new RecurringScheduleEntitiesBuilder(noOfDOMRecurringSchedules, noOfDOWRecurringSchedules).build());

	}

	public ScheduleBuilder setInstanceMaxCount(int max) {
		schedules.setInstance_max_count(max);
		return this;
	}

	public ScheduleBuilder setInstanceMinCount(int min) {
		schedules.setInstance_min_count(min);
		return this;
	}

	public ScheduleBuilder setTimezone(String timeZone) {
		schedules.setTimeZone(timeZone);
		return this;
	}

	public ScheduleBuilder setSpecificDateSchedules(List<SpecificDateScheduleEntity> entities) {
		schedules.setSpecific_date(entities);
		return this;
	}

	public ScheduleBuilder setRecurringSchedules(List<RecurringScheduleEntity> entities) {
		schedules.setRecurring_schedule(entities);
		return this;
	}

	public ApplicationScalingSchedules build() {
		return schedules;
	}
}
