package org.cloudfoundry.autoscaler.cloudservice.api.autoscale;

/*

{
  "app_id":                   <app_id>,
  "instance_min_count":       <min_count>,
  "instance_max_count":       <max_count>,
  
  "metric_type":              <"CPU" or "memory">,
  "stat_type":                <"average" or "max">,
  "stat_window":              <secs>,
  "breach_duration":          <secs>,
  "lower_threshold":          <value>,
  "upper_threshold":          <value>,
  "instance_step_count_down": <instance_count>},
  "instance_step_count_up":   <instance_count>},
  
  "start_time":               <time>,
  "end_time":                 <time>,
  "start_set_num_instances":  <count>,
  "end_set_num_instances":    <count>
} 

*/

public class AppGroupConfig
{
	public static final String METRIC_CPU = "CPU";
	public static final String METRIC_MEM = "memory";
	
	public static final String AGGREGATE_TYPE_AVG = "average";
	public static final String AGGREGATE_TYPE_MAX = "max";

	public String[] appIdList             = {"no_name"};
	
	public String   configId              = appIdList[0];
	public int      instanceMinCount      = 1;
	public int      instanceMaxCount      = 10;

	public String   metricType            = METRIC_CPU;
	public String   statType              = AGGREGATE_TYPE_AVG;
	public int      statWindow            = 120;
	public int      breachDuration        = 60;
	public int      lowerThreshold        = 20;
	public int      upperThreshold        = 70;
	public int      instanceStepCountDown = -1;
	public int      instanceStepCountUp   = 1;
	public int      stepCoolDownSecs      = 20;

	public long     startTime             = 0;
	public long     endTime               = 0;
	public int      startSetNumInstances  = 10;
	public int      endSetNumInstances    = 10;

	
	public String[] getAppIdList() {
		return appIdList.clone();
	}

	public void setAppIdList(String[] appIdList) {
		this.appIdList = appIdList.clone();
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
	}

	public int getInstanceStepCountUp() {
		return instanceStepCountUp;
	}

	public void setInstanceStepCountUp(int instanceStepCountUp) {
		this.instanceStepCountUp = instanceStepCountUp;
	}

	public int getStepCoolDownSecs() {
		return stepCoolDownSecs;
	}

	public void setStepCoolDownSecs(int stepCoolDownSecs) {
		this.stepCoolDownSecs = stepCoolDownSecs;
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
