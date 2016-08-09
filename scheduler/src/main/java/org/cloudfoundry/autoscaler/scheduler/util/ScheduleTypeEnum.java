package org.cloudfoundry.autoscaler.scheduler.util;

/**
 * 
 *
 */
public enum ScheduleTypeEnum {
	SPECIFIC_DATE("S", "Specific_Date", "Specific Date Schedule"), RECURRING("R", "Recurring", "Recurring Schedule");

	private String dbValue;
	String scheduleIdentifier;
	private String description;

	private ScheduleTypeEnum(String dbValue, String scheduleIdentifier, String description) {
		this.dbValue = dbValue;
		this.scheduleIdentifier = scheduleIdentifier;
		this.description = description;
	}

	public String getDbValue() {
		return dbValue;
	}

	public void setDbValue(String dbValue) {
		this.dbValue = dbValue;
	}

	public String getScheduleIdentifier() {
		return scheduleIdentifier;
	}

	public void setScheduleIdentifier(String scheduleIdentifier) {
		this.scheduleIdentifier = scheduleIdentifier;
	}

	public String getDescription() {
		return description;
	}

	public void setDescription(String description) {
		this.description = description;
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
