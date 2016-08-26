package org.cloudfoundry.autoscaler.scheduler.util;

import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Date;
import java.util.List;

import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;

public class SpecificDateScheduleEntitiesBuilder {
	private List<SpecificDateScheduleEntity> specificDateScheduleEntities;

	public SpecificDateScheduleEntitiesBuilder(int noOfSchedules) {
		specificDateScheduleEntities = generateEntities(noOfSchedules);
	}

	public SpecificDateScheduleEntitiesBuilder setTimeZone(String timeZone) {
		if (specificDateScheduleEntities != null) {
			for (SpecificDateScheduleEntity specificDateScheduleEntity : specificDateScheduleEntities) {
				specificDateScheduleEntity.setTimezone(timeZone);
			}
		}
		return this;
	}

	public SpecificDateScheduleEntitiesBuilder setAppid(String appId) {
		if (specificDateScheduleEntities != null) {
			for (SpecificDateScheduleEntity specificDateScheduleEntity : specificDateScheduleEntities) {
				specificDateScheduleEntity.setApp_id(appId);
			}
		}
		return this;
	}

	public SpecificDateScheduleEntitiesBuilder setDefaultInstanceMaxCount(int max) {
		if (specificDateScheduleEntities != null) {
			for (SpecificDateScheduleEntity specificDateScheduleEntity : specificDateScheduleEntities) {
				specificDateScheduleEntity.setDefault_instance_max_count(max);
			}
		}
		return this;

	}

	public SpecificDateScheduleEntitiesBuilder setDefaultInstanceMinCount(int min) {
		if (specificDateScheduleEntities != null) {
			for (SpecificDateScheduleEntity specificDateScheduleEntity : specificDateScheduleEntities) {
				specificDateScheduleEntity.setDefault_instance_min_count(min);
			}
		}
		return this;
	}

	public SpecificDateScheduleEntitiesBuilder setInstanceMaxCount(int pos, int max) {
		specificDateScheduleEntities.get(pos).setInstance_max_count(max);
		return this;
	}

	public SpecificDateScheduleEntitiesBuilder setInstanceMinCount(int pos, int min) {
		specificDateScheduleEntities.get(pos).setInstance_min_count(min);
		return this;
	}

	public SpecificDateScheduleEntitiesBuilder setStartDateTime(int pos, Date date) {
		specificDateScheduleEntities.get(pos).setStart_date_time(date);
		return this;
	}

	public SpecificDateScheduleEntitiesBuilder setEndDateTime(int pos, Date date) {
		specificDateScheduleEntities.get(pos).setEnd_date_time(date);
		return this;
	}

	public List<SpecificDateScheduleEntity> build() {
		return specificDateScheduleEntities;
	}

	private List<SpecificDateScheduleEntity> generateEntities(int noOfEntities) {
		if (noOfEntities <= 0) {
			return null;
		}
		List<SpecificDateScheduleEntity> specificDateScheduleEntities = new ArrayList<>();

		int pos = 0;
		SimpleDateFormat sdf = new SimpleDateFormat(DateHelper.DATE_TIME_FORMAT);
		for (int i = 0; i < noOfEntities; i++) {
			SpecificDateScheduleEntity specificDateScheduleEntity = new SpecificDateScheduleEntity();

			try {
				specificDateScheduleEntity.setStart_date_time(
						sdf.parse(TestDataSetupHelper.getDateString(TestDataSetupHelper.getStartDateTime(), pos, 0)));
				specificDateScheduleEntity.setEnd_date_time(
						sdf.parse(TestDataSetupHelper.getDateString(TestDataSetupHelper.getEndDateTime(), pos, 5)));
			} catch (ParseException e) {
				throw new RuntimeException(e.getMessage());
			}

			specificDateScheduleEntity.setInstance_min_count(i + 5);
			specificDateScheduleEntity.setInstance_max_count(i + 6);
			specificDateScheduleEntities.add(specificDateScheduleEntity);
			pos++;
		}

		return specificDateScheduleEntities;
	}

}
