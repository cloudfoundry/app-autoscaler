package org.cloudfoundry.autoscaler.scheduler.util;

/**
 * @author Fujitsu
 *
 */
public enum JobScheduleTypeEnum {
	CRON("C", "schedule_type_cron"), SIMPLE("S", "schedule_type_simple");

	private String dbValue;
	private String description;

	private JobScheduleTypeEnum(String dbValue, String description) {
		this.dbValue = dbValue;
		this.description = description;
	}

	public String getDbValue() {
		return dbValue;
	}

	public void setDbValue(String dbValue) {
		this.dbValue = dbValue;
	}

	public String getDescription() {
		return description;
	}

	public void setDescription(String description) {
		this.description = description;
	}

	public static JobScheduleTypeEnum getEnum(String str) {
		for (JobScheduleTypeEnum value : values()) {
			if (value.getDbValue().equals(str)) {
				return value;
			}
		}
		throw new IllegalArgumentException("No such a Enum:" + str);
	}

}
