package org.cloudfoundry.autoscaler.scheduler.util;

/**
 * 
 *
 */
public enum JobActionEnum {
	START("Starting", "Start Scaling Job", "_start"), END("Ending", "End Scaling Job", "_end");

	private String status;
	private String description;
	private String jobIdSuffix;

	JobActionEnum(String status, String description, String jobIdSuffix) {
		this.status = status;
		this.description = description;
		this.jobIdSuffix = jobIdSuffix;
	}

	public String getStatus() {
		return status;
	}

	public String getDescription() {
		return description;
	}

	public String getJobIdSuffix() {
		return jobIdSuffix;
	}

}
