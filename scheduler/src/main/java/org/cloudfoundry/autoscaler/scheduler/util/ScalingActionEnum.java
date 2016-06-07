package org.cloudfoundry.autoscaler.scheduler.util;

/**
 * @author Fujitsu
 *
 */
public enum ScalingActionEnum {
	START("S", "start_scaling_action"), END("E", "end_scaling_action");

	private String action;
	private String description;

	private ScalingActionEnum(String dbValue, String description) {
		this.action = dbValue;
		this.description = description;
	}

	public String getAction() {
		return action;
	}

	public void setAction(String action) {
		this.action = action;
	}

	public String getDescription() {
		return description;
	}

	public void setDescription(String description) {
		this.description = description;
	}

}
