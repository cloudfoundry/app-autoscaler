package org.cloudfoundry.autoscaler.api.validation;

import java.text.ParsePosition;
import java.text.SimpleDateFormat;
import java.util.Arrays;
import java.util.Date;
import java.util.HashSet;
import java.util.List;
import java.util.Set;

import javax.validation.constraints.AssertTrue;
import javax.validation.constraints.Min;
import javax.validation.constraints.NotNull;

public class recurringSchedule{

	@NotNull(message="{recurringSchedule.minInstCount.NotNull}")
	@Min(value=1, message="{recurringSchedule.minInstCount.Min}") int minInstCount;

	int maxInstCount = -1;//if maxInstCount is not appears in the parameter, it's set to 0 which automatically fulfill the validation requirement, actual value should be set after validation

	boolean maxInstCount_set = false; //true if maxInstCount is set in json or set false;
	@NotNull(message="{recurringSchedule.startTime.NotNull}")
	private String startTime;

	@NotNull(message="{recurringSchedule.endTime.NotNull}")
	private String endTime;

	@NotNull(message="{recurringSchedule.repeatOn.NotNull}")
	private String repeatOn;

	@AssertTrue(message="{recurringSchedule.isRepeatOnValid.AssertTrue}")
	private boolean isRepeatOnValid() {
		String[] s_values = this.repeatOn.replace("\"", "").replace("[", "").replace("]", "").split(",");
		String[] weekday = {"1", "2", "3", "4", "5", "6", "7"};
	    List<String> weekday_list = Arrays.asList(weekday);
	    Set<String> validValues = new HashSet<String>(weekday_list);

	    List<String> value_list = Arrays.asList(s_values);
	    Set<String> value_set = new HashSet<String>(value_list);

	    if ( s_values.length > value_set.size()) {
			return false;
		}
		for (String s: s_values){
			if(!validValues.contains(s)) {
				return false;
			}
		}
		if ( s_values.length > validValues.size()) {
			return false;
		}
		return true;
	}

	@AssertTrue(message="{recurringSchedule.isTimeValid.AssertTrue}")
	private boolean isTimeValid() {
		try {
			SimpleDateFormat parser = new SimpleDateFormat("HH:mm");
			parser.setLenient(false);
        	ParsePosition position = new ParsePosition(0);
			Date start_time = parser.parse(this.startTime, position);
        	if (( start_time == null) || (position.getIndex() != this.startTime.length()))
        		return false;
        	position = new ParsePosition(0);
			Date end_time = parser.parse(this.endTime, position);
        	if (( end_time == null) || (position.getIndex() != this.endTime.length()))
        		return false;
        	return start_time.before(end_time);
		} catch (Exception e) {
			return false;
		}
	}

	@AssertTrue(message="{recurringSchedule.isInstCountValid.AssertTrue}")
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

	public void setMaxInstCount(int maxInstCount) {
		this.maxInstCount_set = true; //we assume that setMaxInstCount will be called by Jackson when maxInstCount is set in JSON, or this will not be called
		this.maxInstCount = maxInstCount;
	}

	public String getStartTime() {
		return this.startTime;
	}

	public void setStartTime(String startTime) {
		this.startTime = startTime;
	}

	public String getEndTime() {
		return this.endTime;
	}

	public void setEndTime(String endTime) {
		this.endTime = endTime;
	}

	public String getRepeatOn() {
		return this.repeatOn;
	}

	public void setRepeatOn(String repeatOn) {
		this.repeatOn = repeatOn;
	}
}