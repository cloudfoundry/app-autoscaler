package org.cloudfoundry.autoscaler.data.couchdb.document;

import java.io.Serializable;

import org.ektorp.support.TypeDiscriminator;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;


@JsonIgnoreProperties(ignoreUnknown = true)
@TypeDiscriminator ("doc.type=='ScalingHistory'")
public class ScalingHistory extends TypedCouchDbDocument{
	/**
	 * 
	 */
	private static final long serialVersionUID = -5548154609735082945L;
	private String appId;
	private int status;
	private int adjustment; // instance count adjustment
	private int instances;// current instance count
	private long startTime;// start time of scaling
	private long endTime;// end time of scaling
	private ScalingTrigger trigger;// The event that trigger this scaling
	private String errorCode;//the error code of failed scaling
	private String scheduleType;
	private String timeZone;
	private Long scheduleStartTime;
	private Integer dayOfWeek;
	private Integer rawOffset;
	
	public ScalingHistory (){
		super();
	}
	
	public ScalingHistory(String appId,int status, int adjustment, int instances,
			long startTime, long endTime, String metricsName, int threshold,
			String thresholdType, int breachDuration, int triggerType, String errorCode, String scheduleType,String timeZone,Long scheduleStartTime,Integer dayOfWeek) {
		super();
		this.appId = appId;
		this.status = status;
		this.adjustment = adjustment;
		this.instances = instances;
		this.startTime = startTime;
		this.endTime = endTime;
		this.trigger = new ScalingTrigger(metricsName, threshold,
				thresholdType, breachDuration,triggerType);
		this.errorCode = errorCode;
		this.scheduleType = scheduleType;
		this.timeZone = timeZone;
		this.scheduleStartTime = scheduleStartTime;
		this.dayOfWeek = dayOfWeek;
	}

	public String getAppId() {
		return appId;
	}
	public void setAppId(String appId) {
		this.appId = appId;
	}
	public int getStatus() {
		return status;
	}

	public void setStatus(int status) {
		this.status = status;
	}

	public int getAdjustment() {
		return adjustment;
	}

	public void setAdjustment(int adjustment) {
		this.adjustment = adjustment;
	}

	public int getInstances() {
		return instances;
	}

	public void setInstances(int instances) {
		this.instances = instances;
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

	public ScalingTrigger getTrigger() {
		return trigger;
	}

	public void setTrigger(ScalingTrigger trigger) {
		this.trigger = trigger;
	}

	public static class ScalingTrigger implements Serializable{
		/**
		 * 
		 */
		private static final long serialVersionUID = 321323026590016595L;
		private String metrics;
		private int threshold;
		private String thresholdType;
		private int breachDuration;
		private int triggerType;
		public ScalingTrigger(){
			
		}
		public ScalingTrigger(String metrics, int threshold,
				String thresholdType, int breachDuration, int triggerType) {
			super();
			this.metrics = metrics;
			this.threshold = threshold;
			this.thresholdType = thresholdType;
			this.breachDuration = breachDuration;
			this.triggerType = triggerType;
		}
		public String getMetrics() {
			return metrics;
		}
		public void setMetrics(String metrics) {
			this.metrics = metrics;
		}
		public int getThreshold() {
			return threshold;
		}
		public void setThreshold(int threshold) {
			this.threshold = threshold;
		}
		public String getThresholdType() {
			return thresholdType;
		}
		public void setThresholdType(String thresholdType) {
			this.thresholdType = thresholdType;
		}
		public int getBreachDuration() {
			return breachDuration;
		}
		public void setBreachDuration(int breachDuration) {
			this.breachDuration = breachDuration;
		}
		public int getTriggerType() {
			return triggerType;
		}
		public void setTriggerType(int triggerType) {
			this.triggerType = triggerType;
		}
		
	}

	public String getErrorCode() {
		return errorCode;
	}
	public void setErrorCode(String errorCode) {
		this.errorCode = errorCode;
	}

	public String getScheduleType() {
		return scheduleType;
	}

	public void setScheduleType(String scheduleType) {
		this.scheduleType = scheduleType;
	}

	public String getTimeZone() {
		return timeZone;
	}

	public void setTimeZone(String timeZone) {
		this.timeZone = timeZone;
	}

	public Long getScheduleStartTime() {
		return scheduleStartTime;
	}

	public void setScheduleStartTime(Long scheduleStartTime) {
		this.scheduleStartTime = scheduleStartTime;
	}

	public Integer getDayOfWeek() {
		return dayOfWeek;
	}

	public void setDayOfWeek(Integer dayOfWeek) {
		this.dayOfWeek = dayOfWeek;
	}

	public Integer getRawOffset() {
		return rawOffset;
	}

	public void setRawOffset(Integer rawOffset) {
		this.rawOffset = rawOffset;
	}
}
