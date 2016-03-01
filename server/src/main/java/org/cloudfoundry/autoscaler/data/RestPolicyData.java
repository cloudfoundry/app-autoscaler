package org.cloudfoundry.autoscaler.data;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;


@JsonIgnoreProperties(ignoreUnknown = true)
public class RestPolicyData
{
	
	private String   configId              = PolicyData.UNSPECIFIED_STRING;
	
	private int      instanceMinCount      = PolicyData.UNSPECIFIED_INT;
	private int      instanceMaxCount      = PolicyData.UNSPECIFIED_INT;

	private String   metricType            = PolicyData.UNSPECIFIED_STRING;
	private String   statType              = PolicyData.UNSPECIFIED_STRING;
	private int      statWindow            = PolicyData.UNSPECIFIED_INT;
	private int      breachDuration        = PolicyData.UNSPECIFIED_INT;
	private int      lowerThreshold        = PolicyData.UNSPECIFIED_INT;
	private int      upperThreshold        = PolicyData.UNSPECIFIED_INT;
	private int      instanceStepCountDown = PolicyData.UNSPECIFIED_INT;
	private int      instanceStepCountUp   = PolicyData.UNSPECIFIED_INT;
	private int      stepDownCoolDownSecs  = PolicyData.UNSPECIFIED_INT;
	private int      stepUpCoolDownSecs    = PolicyData.UNSPECIFIED_INT;

	private long     startTime             = PolicyData.UNSPECIFIED_LONG;
	private long     endTime               = PolicyData.UNSPECIFIED_LONG;
	private int      startSetNumInstances  = PolicyData.UNSPECIFIED_INT;
	private int      endSetNumInstances    = PolicyData.UNSPECIFIED_INT;

	
	public RestPolicyData()
	{
	}
	
	public PolicyData getAppConfigData()
	{
		PolicyData config = new PolicyData();
		
		config.setInstanceMinCount      ( this.getInstanceMinCount() ); 
		config.setInstanceMaxCount      ( this.getInstanceMaxCount() ); 

		config.setMetricType            ( this.getMetricType() ); 
		config.setStatType              ( this.getStatType() ); 
		config.setStatWindow            ( this.getStatWindow() ); 
		config.setBreachDuration        ( this.getBreachDuration() ); 
		config.setLowerThreshold        ( this.getLowerThreshold() ); 
		config.setUpperThreshold        ( this.getUpperThreshold() ); 
		config.setInstanceStepCountDown ( this.getInstanceStepCountDown() ); 
		config.setInstanceStepCountUp   ( this.getInstanceStepCountUp() ); 
		config.setStepDownCoolDownSecs  ( this.getStepDownCoolDownSecs() ); 
		config.setStepUpCoolDownSecs    ( this.getStepUpCoolDownSecs() ); 

		config.setStartTime             ( this.getStartTime() ); 
		config.setEndTime               ( this.getEndTime() ); 
		config.setStartSetNumInstances  ( this.getStartSetNumInstances() ); 
		config.setEndSetNumInstances    ( this.getEndSetNumInstances() );
		
		return config;
	}
	
	public String getConfigId() {
		return configId;
	}

	public void setConfigId(String configId) {
		this.configId = configId;
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
