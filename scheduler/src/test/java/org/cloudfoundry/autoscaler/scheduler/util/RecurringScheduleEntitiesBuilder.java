package org.cloudfoundry.autoscaler.scheduler.util;

import java.time.LocalDate;
import java.time.LocalTime;
import java.util.ArrayList;
import java.util.List;
import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;

public class RecurringScheduleEntitiesBuilder {
  private int scheduleIndex = 0;
  private List<RecurringScheduleEntity> recurringScheduleEntities;

  public RecurringScheduleEntitiesBuilder(int noOfDomSchedules, int noOfDowSchedules) {
    recurringScheduleEntities = generateEntities(noOfDomSchedules, noOfDowSchedules);
  }

  public RecurringScheduleEntitiesBuilder setTimeZone(String timeZone) {
    if (recurringScheduleEntities != null) {
      for (RecurringScheduleEntity recurringScheduleEntity : recurringScheduleEntities) {
        recurringScheduleEntity.setTimeZone(timeZone);
      }
    }
    return this;
  }

  public RecurringScheduleEntitiesBuilder setDefaultInstanceMaxCount(int max) {
    if (recurringScheduleEntities != null) {
      for (RecurringScheduleEntity recurringScheduleEntity : recurringScheduleEntities) {
        recurringScheduleEntity.setDefaultInstanceMaxCount(max);
      }
    }
    return this;
  }

  public RecurringScheduleEntitiesBuilder setDefaultInstanceMinCount(int min) {
    if (recurringScheduleEntities != null) {
      for (RecurringScheduleEntity recurringScheduleEntity : recurringScheduleEntities) {
        recurringScheduleEntity.setDefaultInstanceMinCount(min);
      }
    }
    return this;
  }

  public RecurringScheduleEntitiesBuilder setInstanceMaxCount(int pos, int max) {
    recurringScheduleEntities.get(pos).setInstanceMaxCount(max);
    return this;
  }

  public RecurringScheduleEntitiesBuilder setInstanceMinCount(int pos, int min) {
    recurringScheduleEntities.get(pos).setInstanceMinCount(min);
    return this;
  }

  public RecurringScheduleEntitiesBuilder setStartTime(int pos, LocalTime time) {
    recurringScheduleEntities.get(pos).setStartTime(time);
    return this;
  }

  public RecurringScheduleEntitiesBuilder setEndTime(int pos, LocalTime time) {
    recurringScheduleEntities.get(pos).setEndTime(time);
    return this;
  }

  public RecurringScheduleEntitiesBuilder setStartDate(int pos, LocalDate date) {
    recurringScheduleEntities.get(pos).setStartDate(date);
    return this;
  }

  public RecurringScheduleEntitiesBuilder setEndDate(int pos, LocalDate date) {
    recurringScheduleEntities.get(pos).setEndDate(date);
    return this;
  }

  public RecurringScheduleEntitiesBuilder setDayOfWeek(int pos, int[] dayOfWeek) {
    recurringScheduleEntities.get(pos).setDaysOfWeek(dayOfWeek);
    return this;
  }

  public RecurringScheduleEntitiesBuilder setDayOfMonth(int pos, int[] dayOfMonth) {
    recurringScheduleEntities.get(pos).setDaysOfMonth(dayOfMonth);
    return this;
  }

  public RecurringScheduleEntitiesBuilder setAppId(String appId) {
    if (recurringScheduleEntities != null) {
      for (RecurringScheduleEntity recurringScheduleEntity : recurringScheduleEntities) {
        recurringScheduleEntity.setAppId(appId);
      }
    }
    return this;
  }

  public RecurringScheduleEntitiesBuilder setScheduleId() {
    long index = 1;
    for (RecurringScheduleEntity recurringScheduleEntity : recurringScheduleEntities) {
      recurringScheduleEntity.setId(index++);
    }

    return this;
  }

  public RecurringScheduleEntitiesBuilder setGuid(String guid) {
    if (recurringScheduleEntities != null) {
      for (RecurringScheduleEntity recurringScheduleEntity : recurringScheduleEntities) {
        recurringScheduleEntity.setGuid(guid);
      }
    }
    return this;
  }

  public List<RecurringScheduleEntity> build() {
    return recurringScheduleEntities;
  }

  private List<RecurringScheduleEntity> generateEntities(
      int noOfDomSchedules, int noOfDowSchedules) {
    List<RecurringScheduleEntity> entities = new ArrayList<>();
    if ((noOfDomSchedules + noOfDowSchedules) == 0) {
      return null;
    }
    entities.addAll(generateDomDowEntities(noOfDomSchedules, false));
    entities.addAll(generateDomDowEntities(noOfDowSchedules, true));

    return entities;
  }

  private List<RecurringScheduleEntity> generateDomDowEntities(int noOfSchedules, boolean isDow) {
    List<RecurringScheduleEntity> recurringScheduleEntities = new ArrayList<>();
    for (int i = 0; i < noOfSchedules; i++) {
      RecurringScheduleEntity recurringScheduleEntity = new RecurringScheduleEntity();

      recurringScheduleEntity.setStartTime(TestDataSetupHelper.getZoneTimeWithOffset(10 * i + 10));

      recurringScheduleEntity.setEndTime(TestDataSetupHelper.getZoneTimeWithOffset(10 * i + 15));

      recurringScheduleEntity.setInstanceMinCount(i + 5);
      recurringScheduleEntity.setInstanceMaxCount(i + 6);
      if (isDow) {
        recurringScheduleEntity.setDaysOfWeek(TestDataSetupHelper.generateDayOfWeek());
      } else {
        recurringScheduleEntity.setDaysOfMonth(TestDataSetupHelper.generateDayOfMonth());
      }
      recurringScheduleEntities.add(recurringScheduleEntity);

      scheduleIndex++;
    }

    return recurringScheduleEntities;
  }
}
