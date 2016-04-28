package org.cloudfoundry.autoscaler.api.validation;

import java.util.Arrays;
import java.util.HashSet;
import java.util.Set;

import javax.validation.Valid;
import javax.validation.constraints.AssertTrue;
import javax.validation.constraints.Max;
import javax.validation.constraints.Min;
import javax.validation.constraints.NotNull;

import org.cloudfoundry.autoscaler.common.Constants;

public class HistoryData {
	@NotNull(message="{HistoryData.appId.NotNull}")
	private String appId;

	@NotNull(message="{HistoryData.status.NotNull}")
	@Min(value=-1, message="{HistoryData.status.Min}")
	@Max(value=3, message="{HistoryData.status.Max}")
	private int status;

	@NotNull(message="{HistoryData.adjustment.NotNull}")
	private int adjustment;

	@NotNull(message="{HistoryData.instances.NotNull}")
	private int instances;

	@NotNull(message="{HistoryData.startTime.NotNull}")
	private long startTime;

	@NotNull(message="{HistoryData.endTime.NotNull}")
	private long endTime;

	@Valid
	private ScalingTrigger trigger;

	@NotNull(message="{HistoryData.timeZone.NotNull}")
	private String timeZone;

	private String errorCode="";

	private String scheduleType="";

	@AssertTrue(message="{HistoryData.isTimeZoneValid.AssertTrue}")
	private boolean isTimeZoneValid() {
		Set<String> timezoneset = new HashSet<String>(Arrays.asList(Constants.timezones));
	    if(timezoneset.contains(this.timeZone))
	    	return true;
	    return false;
	}

    public String getAppId(){
    	return this.appId;
    }

    public void setAppId(String appId) {
    	this.appId = appId;
    }

    public int getStatus() {
    	return this.status;
    }

    public void setStatus(int status){
    	this.status = status;
    }

	public int getAdjustment() {
		return this.adjustment;
	}

	public void setAdjustment(int adjustment) {
		this.adjustment = adjustment;
	}

	public int getInstances() {
		return this.instances;
	}

	public void setInstances(int instances) {
		this.instances = instances;
	}

	public long getStartTime() {
		return this.startTime;
	}

	public void setStartTime(long startTime) {
		this.startTime = startTime;
	}

	public long getEndTime() {
		return this.endTime;
	}

	public void setEndTime(long endTime) {
		this.endTime = endTime;
	}

	public ScalingTrigger getTrigger() {
		return this.trigger;
	}

	public void setTrigger(ScalingTrigger trigger) {
		this.trigger = trigger;
	}

	public String getTimeZone() {
		return this.timeZone;
	}

	public void setTimeZone(String timeZone) {
		this.timeZone = timeZone;
	}

    public String getErrorCode() {
    	return this.errorCode;
    }

    public void setErrorCode(String errorCode) {
    	this.errorCode = errorCode;
    }

	public String getScheduleType() {
		return this.scheduleType;
	}

	public void setScheduleType(String scheduleType) {
		this.scheduleType = scheduleType;
	}

}