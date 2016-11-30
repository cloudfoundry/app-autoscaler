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
				specificDateScheduleEntity.setTimeZone(timeZone);
			}
		}
		return this;
	}

	public SpecificDateScheduleEntitiesBuilder setAppid(String appId) {
		if (specificDateScheduleEntities != null) {
			for (SpecificDateScheduleEntity specificDateScheduleEntity : specificDateScheduleEntities) {
				specificDateScheduleEntity.setAppId(appId);
			}
		}
		return this;
	}

	public SpecificDateScheduleEntitiesBuilder setDefaultInstanceMaxCount(int max) {
		if (specificDateScheduleEntities != null) {
			for (SpecificDateScheduleEntity specificDateScheduleEntity : specificDateScheduleEntities) {
				specificDateScheduleEntity.setDefaultInstanceMaxCount(max);
			}
		}
		return this;

	}

	public SpecificDateScheduleEntitiesBuilder setDefaultInstanceMinCount(int min) {
		if (specificDateScheduleEntities != null) {
			for (SpecificDateScheduleEntity specificDateScheduleEntity : specificDateScheduleEntities) {
				specificDateScheduleEntity.setDefaultInstanceMinCount(min);
			}
		}
		return this;
	}

	public SpecificDateScheduleEntitiesBuilder setInstanceMaxCount(int pos, int max) {
		specificDateScheduleEntities.get(pos).setInstanceMaxCount(max);
		return this;
	}

	public SpecificDateScheduleEntitiesBuilder setInstanceMinCount(int pos, int min) {
		specificDateScheduleEntities.get(pos).setInstanceMinCount(min);
		return this;
	}

	public SpecificDateScheduleEntitiesBuilder setStartDateTime(int pos, Date date) {
		specificDateScheduleEntities.get(pos).setStartDateTime(date);
		return this;
	}

	public SpecificDateScheduleEntitiesBuilder setEndDateTime(int pos, Date date) {
		specificDateScheduleEntities.get(pos).setEndDateTime(date);
		return this;
	}

	public SpecificDateScheduleEntitiesBuilder setScheduleId() {
		long index = 1;
		for (SpecificDateScheduleEntity specificDateScheduleEntity : specificDateScheduleEntities) {
			specificDateScheduleEntity.setId(index++);
		}

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
				specificDateScheduleEntity.setStartDateTime(
						sdf.parse(TestDataSetupHelper.getDateString(TestDataSetupHelper.getStartDateTime(), pos, 0)));
				specificDateScheduleEntity.setEndDateTime(
						sdf.parse(TestDataSetupHelper.getDateString(TestDataSetupHelper.getEndDateTime(), pos, 5)));
			} catch (ParseException e) {
				throw new RuntimeException(e.getMessage());
			}

			specificDateScheduleEntity.setInstanceMinCount(i + 5);
			specificDateScheduleEntity.setInstanceMaxCount(i + 6);
			specificDateScheduleEntities.add(specificDateScheduleEntity);
			pos++;
		}

		return specificDateScheduleEntities;
	}

}
