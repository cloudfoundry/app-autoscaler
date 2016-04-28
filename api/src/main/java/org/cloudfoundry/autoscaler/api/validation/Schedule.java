package org.cloudfoundry.autoscaler.api.validation;

import java.util.Arrays;
import java.util.HashSet;
import java.util.List;
import java.util.Set;

import javax.validation.Valid;
import javax.validation.constraints.AssertTrue;
import javax.validation.constraints.NotNull;

import org.cloudfoundry.autoscaler.common.Constants;

public class Schedule {
		@NotNull(message="{Policy.timezone.NotNull}") //debug
 String timezone;

		@Valid List<recurringSchedule> recurringSchedule;

		@Valid List<specificDate> specificDate;

		@AssertTrue(message="{Schedule.isTimeZoneValid.AssertTrue}") //debug
		private boolean isTimeZoneValid() {
			if ((null != this.timezone) && (this.timezone.trim().length() != 0)) {
				BeanValidation.logger.debug("In schedules timezone is " + this.timezone);//debug
				Set<String> timezoneset = new HashSet<String>(Arrays.asList(Constants.timezones));
			    if(timezoneset.contains(this.timezone))
			    	return true;
			    else
			    	return false;
			}
			else { 
				BeanValidation.logger.debug("timezone is empty in schedules");//debug
                return false; //timezone must be specified in Schedule
            }
		}

		@AssertTrue(message="{Schedule.isScheduleValid.AssertTrue}") //debug
		private boolean isScheduleValid() {
			if (((null == this.recurringSchedule) || (this.recurringSchedule.size() == 0) ) && ((null == this.specificDate) || (this.specificDate.size() == 0)) ){
                return false;  //at least one setting should be exist
            }
			return true;
		}

		public String getTimezone() {
			return this.timezone;
		}

		public void setTimezone(String timezone) {
			this.timezone = timezone;
		}

		public List<recurringSchedule> getRecurringSchedule() {
			return this.recurringSchedule;
		}

		public void setRecurringSchedule(List<recurringSchedule> recurringSchedule) {
			this.recurringSchedule = recurringSchedule;
		}

		public List<specificDate> getSpecificDate() {
			return this.specificDate;
		}

		public void setSpecificDate(List<specificDate> specificDate) {
			this.specificDate = specificDate;
		}

		//logic validation are done in Policy structure
	}