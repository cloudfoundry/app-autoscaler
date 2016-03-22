package org.cloudfoundry.autoscaler.api.validation;

import javax.validation.constraints.Max;
import javax.validation.constraints.Min;
import javax.validation.constraints.NotNull;

public class ScalingTrigger {
	@NotNull( message="{ScalingTrigger.metrics.NotNull}")
	private String metrics;

	@NotNull( message="{ScalingTrigger.threshold.NotNull}")
	private int threshold;

	@NotNull( message="{ScalingTrigger.thresholdType.NotNull}")
	private String thresholdType;

	@NotNull( message="{ScalingTrigger.breachDuration.NotNull}")
	private int breachDuration;

	@NotNull( message="{ScalingTrigger.triggerType.NotNull}")
	@Min(value=0, message="{ScalingTrigger.triggerType.Min}")
	@Max(value=1, message="{ScalingTrigger.triggerType.Max}")
	private int triggerType;


	public String getMetrics() {
		return this.metrics;
	}

	public void setMetrics(String metrics){
		this.metrics = metrics;
	}

	public int getThreshold() {
		return this.threshold;
	}

	public void setThreshold(int threshold) {
		this.threshold = threshold;
	}

	public String getThresholdType() {
		return this.thresholdType;
	}

	public void setThresholdType(String thresholdType) {
		this.thresholdType = thresholdType;
	}

	public int getBreachDuration() {
		return this.breachDuration;
	}

	public void setBreachDuration(int breachDuration) {
		this.breachDuration = breachDuration;
	}

	public int getTriggerType() {
		return this.triggerType;
	}

	public void setTriggerType(int triggerType) {
		this.triggerType = triggerType;
	}
}