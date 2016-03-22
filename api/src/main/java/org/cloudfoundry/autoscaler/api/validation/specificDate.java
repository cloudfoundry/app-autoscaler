package org.cloudfoundry.autoscaler.api.validation;

import java.text.ParsePosition;
import java.text.SimpleDateFormat;
import java.util.Date;

import javax.validation.constraints.AssertTrue;
import javax.validation.constraints.Min;
import javax.validation.constraints.NotNull;

public class specificDate{

	@NotNull(message="{specificDate.minInstCount.NotNull}")
	@Min(value=1, message="{specificDate.minInstCount.Min}") int minInstCount;

	int maxInstCount = -1; //if maxInstCount is not appears in the parameter, it's set to 0 which automatically fulfill the validation requirement, actual value should be set after validation

	boolean maxInstCount_set = false; //true if maxInstCount is set in json or set false;
	@NotNull(message="{specificDate.startDate.NotNull}")
	private String startDate;

	@NotNull(message="{specificDate.startTime.NotNull}")
	private String startTime;

	@NotNull(message="{specificDate.endDate.NotNull}")
	private String endDate;

	@NotNull(message="{specificDate.endTime.NotNull}")
	private String endTime;

	@AssertTrue(message="{specificDate.isDateTimeValid.AssertTrue}")
	private boolean isDateTimeValid() {
		try {
			SimpleDateFormat dateformat = new SimpleDateFormat("yyyy-MM-dd HH:mm");
			dateformat.setLenient(false);
			ParsePosition position = new ParsePosition(0);
        	String firstDateTime = this.startDate + " " + this.startTime;
        	String secondDateTime = this.endDate + " " + this.endTime;
        	Date first = dateformat.parse(firstDateTime, position);
        	if (( first == null) || (position.getIndex() != firstDateTime.length()))
        		return false;
        	position = new ParsePosition(0);
        	Date second = dateformat.parse(secondDateTime, position);
        	if (( second == null) || (position.getIndex() != secondDateTime.length()))
        		return false;
        	return first.before(second);
		} catch (Exception e) {
			BeanValidation.logger.info(e.getMessage());
			return false;
		}
	}

	@AssertTrue(message="{specificDate.isInstCountValid.AssertTrue}")
	private boolean isInstCountValid() {
		if (this.maxInstCount > 0) { //maxInstCount is set
			if (this.minInstCount > this.maxInstCount) {
			    	return false;
			}
		}
	    return true;
	}

    public int getMinInstCount() {
    	return this.minInstCount;
    }

    public void setMinInstCount(int minInstCount) {
    	this.minInstCount = minInstCount;
    }

	public int getMaxInstCount() {
		return this.maxInstCount;
	}

	public void setMaxInstCount(int maxInstCount){
		this.maxInstCount_set = true; //we assume that setMaxInstCount will be called by Jackson when maxInstCount is set in JSON, or this will not be called
		this.maxInstCount = maxInstCount;
	}

	public String getStartDate() {
		return this.startDate;
	}

	public void setStartDate(String startDate) {
		this.startDate = startDate;
	}

	public String getStartTime() {
		return this.startTime;
	}

	public void setStartTime(String startTime){
		this.startTime = startTime;
	}

	public String getEndDate() {
		return this.endDate;
	}

	public void setEndDate(String endDate){
		this.endDate = endDate;
	}

	public String getEndTime() {
		return this.endTime;
	}

	public void setEndTime(String endTime) {
		this.endTime = endTime;
	}
}