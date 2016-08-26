package org.cloudfoundry.autoscaler.scheduler.util;

import java.sql.Time;
import java.util.ArrayList;
import java.util.Date;
import java.util.List;

import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;

public class RecurringScheduleEntitiesBuilder {
	private int scheduleIndex = 0;
	private List<RecurringScheduleEntity> recurringScheduleEntities;

	public RecurringScheduleEntitiesBuilder(int noOfDOMSchedules, int noOfDOWSchedules) {
		recurringScheduleEntities = generateEntities(noOfDOMSchedules, noOfDOWSchedules);
	}

	public RecurringScheduleEntitiesBuilder setTimeZone(String timeZone) {
		if (recurringScheduleEntities != null) {
			for (RecurringScheduleEntity recurringScheduleEntity : recurringScheduleEntities) {
				recurringScheduleEntity.setTimezone(timeZone);
			}
		}
		return this;
	}

	public RecurringScheduleEntitiesBuilder setDefaultInstanceMaxCount(int max) {
		if (recurringScheduleEntities != null) {
			for (RecurringScheduleEntity recurringScheduleEntity : recurringScheduleEntities) {
				recurringScheduleEntity.setDefault_instance_max_count(max);
			}
		}
		return this;
	}

	public RecurringScheduleEntitiesBuilder setDefaultInstanceMinCount(int min) {
		if (recurringScheduleEntities != null) {
			for (RecurringScheduleEntity recurringScheduleEntity : recurringScheduleEntities) {
				recurringScheduleEntity.setDefault_instance_min_count(min);
			}
		}
		return this;
	}

	public RecurringScheduleEntitiesBuilder setInstanceMaxCount(int pos, int max) {
		recurringScheduleEntities.get(pos).setInstance_max_count(max);
		return this;
	}

	public RecurringScheduleEntitiesBuilder setInstanceMinCount(int pos, int min) {
		recurringScheduleEntities.get(pos).setInstance_min_count(min);
		return this;
	}

	public RecurringScheduleEntitiesBuilder setStartTime(int pos, Time time) {
		recurringScheduleEntities.get(pos).setStart_time(time);
		return this;
	}

	public RecurringScheduleEntitiesBuilder setEndTime(int pos, Time time) {
		recurringScheduleEntities.get(pos).setEnd_time(time);
		return this;
	}

	public RecurringScheduleEntitiesBuilder setStartDate(int pos, Date date) {
		recurringScheduleEntities.get(pos).setStart_date(date);
		return this;
	}

	public RecurringScheduleEntitiesBuilder setEndDate(int pos, Date date) {
		recurringScheduleEntities.get(pos).setEnd_date(date);
		return this;
	}

	public RecurringScheduleEntitiesBuilder setDayOfWeek(int pos, int[] dayOfWeek) {
		recurringScheduleEntities.get(pos).setDays_of_week(dayOfWeek);
		return this;
	}

	public RecurringScheduleEntitiesBuilder setDayOfMonth(int pos, int[] dayOfMonth) {
		recurringScheduleEntities.get(pos).setDay_of_month(dayOfMonth);
		return this;
	}

	public RecurringScheduleEntitiesBuilder setAppId(String appId) {
		if (recurringScheduleEntities != null) {
			for (RecurringScheduleEntity recurringScheduleEntity : recurringScheduleEntities) {
				recurringScheduleEntity.setApp_id(appId);
			}
		}
		return this;
	}

	public List<RecurringScheduleEntity> build() {
		return recurringScheduleEntities;
	}

	private List<RecurringScheduleEntity> generateEntities(int noOfDOMSchedules, int noOfDOWSchedules) {
		List<RecurringScheduleEntity> entities = new ArrayList<>();
		if ((noOfDOMSchedules + noOfDOWSchedules) == 0) {
			return null;
		}

		entities.addAll(generateDOM_DOWEntities(noOfDOMSchedules, false));
		entities.addAll(generateDOM_DOWEntities(noOfDOWSchedules, true));

		return entities;
	}

	private List<RecurringScheduleEntity> generateDOM_DOWEntities(int noOfSchedules, boolean isDow) {
		List<RecurringScheduleEntity> recurringScheduleEntities = new ArrayList<>();
		for (int i = 0; i < noOfSchedules; i++) {
			RecurringScheduleEntity recurringScheduleEntity = new RecurringScheduleEntity();
			recurringScheduleEntity.setStart_time(TestDataSetupHelper.getTime(TestDataSetupHelper.getStarTime(), scheduleIndex, 0));
			recurringScheduleEntity.setEnd_time(TestDataSetupHelper.getTime(TestDataSetupHelper.getEndTime(), scheduleIndex, 5));

			recurringScheduleEntity.setInstance_min_count(i + 5);
			recurringScheduleEntity.setInstance_max_count(i + 6);
			if (isDow)
				recurringScheduleEntity.setDays_of_week(TestDataSetupHelper.generateDayOfWeek());
			else
				recurringScheduleEntity.setDay_of_month(TestDataSetupHelper.generateDayOfMonth());
			recurringScheduleEntities.add(recurringScheduleEntity);

			scheduleIndex++;
		}

		return recurringScheduleEntities;
	}

}
