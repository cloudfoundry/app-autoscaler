package org.cloudfoundry.autoscaler.scheduler.util;

/** */
public enum ScheduleTypeEnum {
  SPECIFIC_DATE("S", "Specific_Date", "Specific Date Schedule"),
  RECURRING("R", "Recurring", "Recurring Schedule");

  private String dbValue;
  String scheduleIdentifier;
  private String description;

  ScheduleTypeEnum(String dbValue, String scheduleIdentifier, String description) {
    this.dbValue = dbValue;
    this.scheduleIdentifier = scheduleIdentifier;
    this.description = description;
  }

  public String getDbValue() {
    return dbValue;
  }

  public String getScheduleIdentifier() {
    return scheduleIdentifier;
  }

  public String getDescription() {
    return description;
  }

  public static ScheduleTypeEnum getEnum(String str) {
    for (ScheduleTypeEnum value : values()) {
      if (value.getDbValue().equals(str)) {
        return value;
      }
    }
    throw new IllegalArgumentException("No such a Enum:" + str);
  }
}
