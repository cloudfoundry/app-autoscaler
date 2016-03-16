package org.cloudfoundry.autoscaler.bean;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;


@JsonIgnoreProperties(ignoreUnknown = true)
public class AutoScalerPolicyTrigger
{
	
	public static final String METRIC_CPU         = "CPU";
	public static final String METRIC_MEM         = "memory";
	
	public static final String AGGREGATE_TYPE_AVG = "average";
	public static final String AGGREGATE_TYPE_MAX = "max";

	public static final int    UNSPECIFIED_INT    = Integer.MAX_VALUE; // can't use values like 0 or -1
	public static final long   UNSPECIFIED_LONG   = Long.MAX_VALUE;    // can't use values like 0 or -1
	public static final String UNSPECIFIED_STRING = null;              // don't change to anything else (see isUnspecified())
	public static final String TriggerId_LowerThreshold = "lower";
	public static final String TriggerId_UpperThreshold = "upper";
	public static final String ADJUSTMENT_CHANGE_CAPACITY = "changeCapacity";
	public static final String ADJUSTMENT_CHANGE_PERCENTAGE = "changePercentage";
	private String   metricType            = null;
	private String   statType              = AGGREGATE_TYPE_AVG;
	private int      statWindow            = 120;
	private int      breachDuration        = 60;
	private int      lowerThreshold        = 20;
	private int      upperThreshold        = 70;
	private int      instanceStepCountDown = 1;
	private int      instanceStepCountUp   = 2;
	private int      stepDownCoolDownSecs  = 10;
	private int      stepUpCoolDownSecs    = 200;

	private long     startTime             = 0;
	private long     endTime               = 0;
	private int      startSetNumInstances  = 10;
	private int      endSetNumInstances    = 10;
	private String   unit = "percent";
	@JsonProperty("scaleInAdjustment")
	private String scaleInAdjustmentType; //adjustment type, can be changeCapacity or changePercentage
	@JsonProperty("scaleOutAdjustment")
	private String scaleOutAdjustmentType; //adjustment type, can be changeCapacity or changePercentage
	
	public AutoScalerPolicyTrigger()
	{
	}
	
	public void setUnspecifiedValues(AutoScalerPolicyTrigger trigger)
	{

		if (isUnspecified(metricType))            metricType            = trigger.getMetricType(); 
		if (isUnspecified(statType))              statType              = trigger.getStatType(); 
		if (isUnspecified(statWindow))            statWindow            = trigger.getStatWindow(); 
		if (isUnspecified(breachDuration))        breachDuration        = trigger.getBreachDuration(); 
		if (isUnspecified(lowerThreshold))        lowerThreshold        = trigger.getLowerThreshold(); 
		if (isUnspecified(upperThreshold))        upperThreshold        = trigger.getUpperThreshold(); 
		if (isUnspecified(instanceStepCountDown)) instanceStepCountDown = trigger.getInstanceStepCountDown(); 
		if (isUnspecified(instanceStepCountUp))   instanceStepCountUp   = trigger.getInstanceStepCountUp(); 
		if (isUnspecified(stepDownCoolDownSecs))  stepDownCoolDownSecs  = trigger.getStepDownCoolDownSecs(); 
		if (isUnspecified(stepUpCoolDownSecs))    stepUpCoolDownSecs    = trigger.getStepUpCoolDownSecs(); 

		if (isUnspecified(startTime))             startTime             = trigger.getStartTime(); 
		if (isUnspecified(endTime))               endTime               = trigger.getEndTime(); 
		if (isUnspecified(startSetNumInstances))  startSetNumInstances  = trigger.getStartSetNumInstances(); 
		if (isUnspecified(endSetNumInstances))    endSetNumInstances    = trigger.getEndSetNumInstances();
	}
	
	private static boolean isUnspecified(int val) {
		return (val == UNSPECIFIED_INT);
	}
	
	private static boolean isUnspecified(long val) {
		return (val == UNSPECIFIED_LONG);
	}
	
	private static boolean isUnspecified(String val) {
		return (val == UNSPECIFIED_STRING);
	}

	public String getMetricType() {
		return metricType;
	}

	public void setMetricType(String metricType) {
		this.metricType = metricType;
	}

	public String getStatType() {
		return statType;
	}

	public void setStatType(String statType) {
		this.statType = statType;
	}

	public int getStatWindow() {
		return statWindow;
	}

	public void setStatWindow(int statWindow) {
		this.statWindow = statWindow;
	}

	public int getBreachDuration() {
		return breachDuration;
	}

	public void setBreachDuration(int breachDuration) {
		this.breachDuration = breachDuration;
	}

	public int getLowerThreshold() {
		return lowerThreshold;
	}

	public void setLowerThreshold(int lowerThreshold) {
		this.lowerThreshold = lowerThreshold;
	}

	public int getUpperThreshold() {
		return upperThreshold;
	}

	public void setUpperThreshold(int upperThreshold) {
		this.upperThreshold = upperThreshold;
	}

	public int getInstanceStepCountDown() {
		return instanceStepCountDown;
	}

	public void setInstanceStepCountDown(int instanceStepCountDown) {
		this.instanceStepCountDown = instanceStepCountDown;
		if (this.instanceStepCountDown > 0) {
			this.instanceStepCountDown = -this.instanceStepCountDown;
		}
	}

	public int getInstanceStepCountUp() {
		return instanceStepCountUp;
	}

	public void setInstanceStepCountUp(int instanceStepCountUp) {
		this.instanceStepCountUp = instanceStepCountUp;
	}

	public int getStepDownCoolDownSecs() {
		return stepDownCoolDownSecs;
	}

	public void setStepDownCoolDownSecs(int stepDownCoolDownSecs) {
		this.stepDownCoolDownSecs = stepDownCoolDownSecs;
	}

	public int getStepUpCoolDownSecs() {
		return stepUpCoolDownSecs;
	}

	public void setStepUpCoolDownSecs(int stepUpCoolDownSecs) {
		this.stepUpCoolDownSecs = stepUpCoolDownSecs;
	}

	public long getStartTime() {
		return startTime;
	}

	public void setStartTime(long startTime) {
		this.startTime = startTime;
	}

	public long getEndTime() {
		return endTime;
	}

	public void setEndTime(long endTime) {
		this.endTime = endTime;
	}

	public int getStartSetNumInstances() {
		return startSetNumInstances;
	}

	public void setStartSetNumInstances(int startSetNumInstances) {
		this.startSetNumInstances = startSetNumInstances;
	}

	public int getEndSetNumInstances() {
		return endSetNumInstances;
	}

	public void setEndSetNumInstances(int endSetNumInstances) {
		this.endSetNumInstances = endSetNumInstances;
	}

	public String getUnit() {
		return unit;
	}

	public void setUnit(String unit) {
		this.unit = unit;
	}

	public String getScaleInAdjustmentType() {
		return scaleInAdjustmentType;
	}

	public void setScaleInAdjustmentType(String scaleInadjustmentType) {
		this.scaleInAdjustmentType = scaleInadjustmentType;
	}

	public String getScaleOutAdjustmentType() {
		return scaleOutAdjustmentType;
	}

	public void setScaleOutAdjustmentType(String scaleOutadjustmentType) {
		this.scaleOutAdjustmentType = scaleOutadjustmentType;
	}
  
}
