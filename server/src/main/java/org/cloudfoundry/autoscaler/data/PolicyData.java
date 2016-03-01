package org.cloudfoundry.autoscaler.data;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;


@JsonIgnoreProperties(ignoreUnknown = true)
public class PolicyData
{
	
	public static final String METRIC_CPU         = "CPU";
	public static final String METRIC_MEM         = "memory";
	
	public static final String AGGREGATE_TYPE_AVG = "average";
	public static final String AGGREGATE_TYPE_MAX = "max";

	public static final int    UNSPECIFIED_INT    = Integer.MAX_VALUE; // can't use values like 0 or -1
	public static final long   UNSPECIFIED_LONG   = Long.MAX_VALUE;    // can't use values like 0 or -1
	public static final String UNSPECIFIED_STRING = null;              // don't change to anything else (see isUnspecified())
	
	
	private int      instanceMinCount      = UNSPECIFIED_INT;
	private int      instanceMaxCount      = UNSPECIFIED_INT;

	private String   metricType            = UNSPECIFIED_STRING;
	private String   statType              = UNSPECIFIED_STRING;
	private int      statWindow            = UNSPECIFIED_INT;
	private int      breachDuration        = UNSPECIFIED_INT;
	private int      lowerThreshold        = UNSPECIFIED_INT;
	private int      upperThreshold        = UNSPECIFIED_INT;
	private int      instanceStepCountDown = UNSPECIFIED_INT;
	private int      instanceStepCountUp   = UNSPECIFIED_INT;
	private int      stepDownCoolDownSecs  = UNSPECIFIED_INT;
	private int      stepUpCoolDownSecs    = UNSPECIFIED_INT;

	private long     startTime             = UNSPECIFIED_LONG;
	private long     endTime               = UNSPECIFIED_LONG;
	private int      startSetNumInstances  = UNSPECIFIED_INT;
	private int      endSetNumInstances    = UNSPECIFIED_INT;

	
	public PolicyData()
	{
	}
	
	public void setUnspecifiedValues(PolicyData config)
	{
		if (isUnspecified(instanceMinCount))      instanceMinCount      = config.getInstanceMinCount(); 
		if (isUnspecified(instanceMaxCount))      instanceMaxCount      = config.getInstanceMaxCount(); 

		if (isUnspecified(metricType))            metricType            = config.getMetricType(); 
		if (isUnspecified(statType))              statType              = config.getStatType(); 
		if (isUnspecified(statWindow))            statWindow            = config.getStatWindow(); 
		if (isUnspecified(breachDuration))        breachDuration        = config.getBreachDuration(); 
		if (isUnspecified(lowerThreshold))        lowerThreshold        = config.getLowerThreshold(); 
		if (isUnspecified(upperThreshold))        upperThreshold        = config.getUpperThreshold(); 
		if (isUnspecified(instanceStepCountDown)) instanceStepCountDown = config.getInstanceStepCountDown(); 
		if (isUnspecified(instanceStepCountUp))   instanceStepCountUp   = config.getInstanceStepCountUp(); 
		if (isUnspecified(stepDownCoolDownSecs))  stepDownCoolDownSecs  = config.getStepDownCoolDownSecs(); 
		if (isUnspecified(stepUpCoolDownSecs))    stepUpCoolDownSecs    = config.getStepUpCoolDownSecs(); 

		if (isUnspecified(startTime))             startTime             = config.getStartTime(); 
		if (isUnspecified(endTime))               endTime               = config.getEndTime(); 
		if (isUnspecified(startSetNumInstances))  startSetNumInstances  = config.getStartSetNumInstances(); 
		if (isUnspecified(endSetNumInstances))    endSetNumInstances    = config.getEndSetNumInstances();
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
	
	public int getInstanceMinCount() {
		return instanceMinCount;
	}

	public void setInstanceMinCount(int instanceMinCount) {
		this.instanceMinCount = instanceMinCount;
	}

	public int getInstanceMaxCount() {
		return instanceMaxCount;
	}

	public void setInstanceMaxCount(int instanceMaxCount) {
		this.instanceMaxCount = instanceMaxCount;
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
  
}
