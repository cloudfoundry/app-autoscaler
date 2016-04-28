package org.cloudfoundry.autoscaler.api.validation;


import javax.validation.constraints.AssertTrue;
import javax.validation.constraints.NotNull;

import org.cloudfoundry.autoscaler.common.Constants;

public class PolicyTrigger {

	@NotNull(message="{PolicyTrigger.metricType.NotNull}") String   metricType            = null;

	//the following NotNull does not work as Jackson will alway set 0 if not present
	@NotNull(message="{PolicyTrigger.statWindow.NotNull}")
	private int      statWindow = getTriggerDefaultInt("statWindow");

	@NotNull(message="{PolicyTrigger.breachDuration.NotNull}")
	private int      breachDuration = getTriggerDefaultInt("breachDuration");

	@NotNull(message="{PolicyTrigger.lowerThreshold.NotNull}")
	private int      lowerThreshold = -1;//getTriggerDefaultInt("lowerThreshold");
	private boolean lowerThreshold_set = false;

	@NotNull(message="{PolicyTrigger.upperThreshold.NotNull}")
	private int      upperThreshold = -1;//getTriggerDefaultInt("upperThreshold");
	private boolean upperThreshold_set = false;

	@NotNull(message="{PolicyTrigger.instanceStepCountDown.NotNull}") int      instanceStepCountDown = getTriggerDefaultInt("instanceStepCountDown");

	@NotNull(message="{PolicyTrigger.instanceStepCountUp.NotNull}") int      instanceStepCountUp = getTriggerDefaultInt("instanceStepCountUp");

	@NotNull(message="{PolicyTrigger.stepDownCoolDownSecs.NotNull}")
	private int      stepDownCoolDownSecs = getTriggerDefaultInt("stepDownCoolDownSecs");

	@NotNull(message="{PolicyTrigger.stepUpCoolDownSecs.NotNull}")
	private int      stepUpCoolDownSecs    = getTriggerDefaultInt("stepUpCoolDownSecs");

	public int getTriggerDefaultInt(String key) {
		return Constants.getTriggerDefaultInt(key);
	}

	@AssertTrue(message="{PolicyTrigger.isThresholdValid.AssertTrue}")
	private boolean isThresholdValid() {
	    return this.lowerThreshold <= this.upperThreshold; //whatever metricType is, this will always hold
	}


	public String  getMetricType() {
		return this.metricType;
	}

	public void setMetricType(String metricType) {
		this.metricType = metricType;
	}

	public int getStatWindow() {
		return this.statWindow;
	}

	public void setStatWindow(int statWindow) {
		this.statWindow = statWindow;
	}

	public int getBreachDuration() {
		return this.breachDuration;
	}

	public void setBreachDuration(int breachDuration) {
		this.breachDuration = breachDuration;
	}

	public int getLowerThreshold() {
		return this.lowerThreshold;
	}

	public void setLowerThreshold(int lowerThreshold) {
		setLowerThreshold_set(true);
		this.lowerThreshold = lowerThreshold;
	}
	
	public boolean getLowerThreshold_set() {
		return this.lowerThreshold_set;
	}
	
    public void setLowerThreshold_set(boolean lowerThreshold_set) {
    	this.lowerThreshold_set = lowerThreshold_set;
    }
    
	public int getUpperThreshold() {
		return this.upperThreshold;
	}

	public void setUpperThreshold(int upperThreshold) {
		setUpperThreshold_set(true);
		this.upperThreshold = upperThreshold;
	}

	public boolean getUpperThreshold_set() {
		return this.upperThreshold_set;
	}
	
	public void setUpperThreshold_set(boolean upperThreshold_set) {
		this.upperThreshold_set = upperThreshold_set;
	}
	
	public int getInstanceStepCountDown() {
		return this.instanceStepCountDown;
	}

	public void setInstanceStepCountDown(int instanceStepCountDown) {
		this.instanceStepCountDown = instanceStepCountDown;
	}

	public int getInstanceStepCountUp() {
		return this.instanceStepCountUp;
	}

	public void setInstanceStepCountUp(int instanceStepCountUp) {
		this.instanceStepCountUp = instanceStepCountUp;
	}

	public int getStepDownCoolDownSecs() {
		return this.stepDownCoolDownSecs;
	}

	public void setStepDownCoolDownSecs(int stepDownCoolDownSecs) {
		this.stepDownCoolDownSecs = stepDownCoolDownSecs;
	}

	public int getStepUpCoolDownSecs() {
		return this.stepUpCoolDownSecs;
	}

	public void setStepUpCoolDownSecs(int stepUpCoolDownSecs) {
		this.stepUpCoolDownSecs = stepUpCoolDownSecs;
	}
}