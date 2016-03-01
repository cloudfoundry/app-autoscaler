package org.cloudfoundry.autoscaler.api.util;

import java.util.Arrays;
import java.util.Collections;
import java.util.Comparator;
import java.util.HashSet;
import java.util.LinkedList;
import java.util.List;
import java.util.ArrayList;
import java.util.Locale;
import java.util.Map;
import java.util.HashMap;
import java.util.Map.Entry;
import java.util.Set;
import java.util.Iterator;
import java.util.Date;
import java.text.ParsePosition;
import java.text.SimpleDateFormat;
import java.io.IOException;
import java.lang.reflect.Method;

import javax.servlet.http.HttpServletRequest;
import javax.validation.constraints.NotNull;
import javax.validation.constraints.Min;
import javax.validation.constraints.Max;
import javax.validation.constraints.AssertTrue;
import javax.validation.Valid;
import javax.validation.MessageInterpolator;
import javax.validation.ValidatorFactory;
import javax.validation.Validation;
import javax.validation.Validator;
import javax.validation.ConstraintViolation;

import org.apache.log4j.Logger;

//import org.codehaus.jackson.JsonParseException;
//import org.codehaus.jackson.map.JsonMappingException;
//import org.codehaus.jackson.map.ObjectMapper;
//import org.codehaus.jackson.map.DeserializationConfig;
//import org.codehaus.jackson.type.JavaType;
//import org.codehaus.jackson.JsonNode;

import org.hibernate.validator.cfg.ConstraintMapping;
import org.hibernate.validator.HibernateValidatorConfiguration;
import org.hibernate.validator.cfg.defs.MinDef;
import org.hibernate.validator.cfg.defs.MaxDef;
import org.hibernate.validator.cfg.defs.NotNullDef;
import org.hibernate.validator.cfg.defs.ScriptAssertDef;



import java.lang.annotation.ElementType;

import org.hibernate.validator.HibernateValidator;





















import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.node.ArrayNode;
import com.fasterxml.jackson.databind.node.JsonNodeFactory;
import com.fasterxml.jackson.databind.node.ObjectNode;
import com.fasterxml.jackson.databind.JavaType;
import com.fasterxml.jackson.databind.JsonMappingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.fasterxml.jackson.annotation.JsonCreator;
import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.core.JsonParseException;
import com.fasterxml.jackson.core.JsonProcessingException;
import org.cloudfoundry.autoscaler.api.util.LocaleSpecificMessageInterpolator;
import  org.cloudfoundry.autoscaler.api.Constants;

public class ProtoType {
	private static final String CLASS_NAME = ProtoType.class.getName();
	private static final Logger logger     = Logger.getLogger(CLASS_NAME);
	//private static final ObjectMapper mapper = new ObjectMapper().configure(DeserializationConfig.Feature.FAIL_ON_UNKNOWN_PROPERTIES, false);
	private static final ObjectMapper new_mapper = new ObjectMapper().configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false)
			                                                         .configure(DeserializationFeature.ACCEPT_FLOAT_AS_INT, false)
			                                                         .configure(DeserializationFeature.FAIL_ON_NULL_FOR_PRIMITIVES, true);
			                                                        // .configure(SerializationFeature.INDENT_OUTPUT, true);
			                                                        // .configure(DeserializationFeature.FAIL_ON_MISSING_CREATOR_PROPERTIES, true);

	public static Map<String, String> Obj2Map(Object obj) {
		Map<String, String> hashMap = new HashMap<String, String>();
		try {
			Class c = obj.getClass();
			Method m[] = c.getDeclaredMethods();
			for (int i = 0; i < m.length; i++) {
			    if (m[i].getName().indexOf("get")==0) {
			    	String field_name = m[i].getName().replace("get", "");
			    	field_name = field_name.substring(0,1).toLowerCase() + field_name.substring(1);
			    	Object field_value = m[i].invoke(obj, new Object[0]);
			    	if (field_value != null) {
			    		if (field_value instanceof String)
			    			hashMap.put(field_name, field_value.toString());
			    		else
			    		//String field_value_type = field_value.getClass().getName();
			                hashMap.put(field_name, new_mapper.writeValueAsString(field_value));
			    	}
			        else
			        	hashMap.put(field_name, "");
			    }
			}
	    } catch (Exception e) {
	    	logger.info("error in getserverinfo: " + e.getMessage());
	    }
	    return hashMap;
	}

    //compare based on String or time in format "HH:mm"
	static class MapComparator1 implements Comparator<Map<String, String>>
	{
		private String key;
	    private String keyType;

	    public MapComparator1(String key, String keyType)
	    {
	    	this.key = key;
	        this.keyType = keyType;
	    }

	    public int compare(Map<String, String> first,
	                       Map<String, String> second)
	    {
	        // TODO: Null checking, both for maps and values
	        String firstValue = first.get(key);
	        String secondValue = second.get(key);
	        if (keyType.equals("String")){
	        	return firstValue.compareTo(secondValue);
	        }
	        else if (keyType.equals("long")){
	        	long first_value = Long.valueOf(firstValue);
	        	long second_value = Long.valueOf(secondValue);

	        	if (first_value < second_value)
	        		return -1;
	        	else if(first_value == second_value)
	        		return 0;
	        	else
	        		return 1;
	        }
	        else if (keyType.equals("Time")){
	        	String[] first_value = firstValue.split(":");
				String[] second_value = secondValue.split(":");
				if((Integer.parseInt(first_value[0]) == Integer.parseInt(second_value[0])) &&
					(Integer.parseInt(first_value[1]) == Integer.parseInt(second_value[1]))) {
					return 0;
				}
				else if((Integer.parseInt(first_value[0]) < Integer.parseInt(second_value[0])) ||
						((Integer.parseInt(first_value[0]) == Integer.parseInt(second_value[0])) &&
						 (Integer.parseInt(first_value[1]) < Integer.parseInt(second_value[1])))) {
						return -1;
				}
				else {
					return 1;
				}
	        } else if(keyType.equals("Date")){
	        	SimpleDateFormat dateformat = new SimpleDateFormat("yyyy-MM-dd");
	        	dateformat.setLenient(false);
	        	ParsePosition position = new ParsePosition(0);
 	        	Date firstdate = dateformat.parse(firstValue, position);
 	        	if (( firstdate == null) || (position.getIndex() != firstValue.length()))
 	        		return -2;
 	        	position = new ParsePosition(0);
 	        	Date seconddate = dateformat.parse(secondValue, position);
 	        	if (( seconddate == null) || (position.getIndex() != secondValue.length()))
 	        		return -2;
	        	try {
	        	    if(firstdate.before(seconddate)){
	        		    return -1;
	        	    } else if (firstdate.equals(seconddate)){
	        		    return 0;
	        	    } else {
	        	    	return 1;
	        	    }
	        	} catch (Exception e){
	        		return -2;
	        	}
	        }

	        return -2;
	    }
	}

	//Datetime compare in format yyyy-MM-dd HH:mm
	static class MapComparator2 implements Comparator<Map<String, String>>
	{
		private String key1;
		private String key2;
	    private String keyType;

	    public MapComparator2(String key1, String key2, String keyType)
	    {
	    	this.key1 = key1;
	    	this.key2 = key2;
	        this.keyType = keyType;
	    }

	    public int compare(Map<String, String> first,
	                       Map<String, String> second)
	    {
	        // TODO: Null checking, both for maps and values
	        String firstValue_key1 = first.get(key1);
	        String firstValue_key2 = first.get(key2);
	        String secondValue_key1 = second.get(key1);
	        String secondValue_key2 = second.get(key2);
	        if(keyType.equals("DateTime")){ //assume date in key1 and time in key2
	        	int datecompare, timecompare;

	        	SimpleDateFormat dateformat = new SimpleDateFormat("yyyy-MM-dd HH:mm");
	        	dateformat.setLenient(false);
 	        	String firstDateTime = firstValue_key1 + " " + firstValue_key2;
 	        	String secondDateTime = secondValue_key1 + " " + secondValue_key2;
 	        	ParsePosition position = new ParsePosition(0);
 	        	Date firstdate = dateformat.parse(firstDateTime, position);
 	        	if (( firstdate == null) || (position.getIndex() != firstDateTime.length()))
 	        		return -2;
 	        	position = new ParsePosition(0);
 	        	Date seconddate = dateformat.parse(secondDateTime, position);
 	        	if (( seconddate == null) || (position.getIndex() != secondDateTime.length()))
 	        		return -2;
 	        	try {
	 	        	if(firstdate.before(seconddate))
	 	        		return -1;
	 	        	else if (firstdate.equals(seconddate))
	 	        		return 0;
	 	        	else
	 	        		return 1;
 	        	} catch (Exception e){
 	        		return -2;
 	        	}
	        }

	        return -2;
	    }
	}
	//Timestamp or other long type compare
	static class MapComparator3 implements Comparator<Map<String, Object>>
	{
		private String key;
	    private String keyType;

	    public MapComparator3(String key, String keyType)
	    {
	    	this.key = key;
	        this.keyType = keyType;
	    }

	    public int compare(Map<String, Object> first,
	                       Map<String, Object> second)
	    {
	        // TODO: Null checking, both for maps and values
	        Object firstValue = first.get(key);
	        Object secondValue = second.get(key);
	        if (keyType.equals("long")){
	        	long first_value = Long.valueOf(firstValue.toString());
	        	long second_value = Long.valueOf(secondValue.toString());

	        	if (first_value < second_value)
	        		return -1;
	        	else if(first_value == second_value)
	        		return 0;
	        	else
	        		return 1;

	        }

	        return -2;
	    }
	}

	public static class JsonObjectComparator implements Comparator<JsonNode> {
		private final String fieldName;
		private final String fieldType;

		public JsonObjectComparator(String fieldName, String fieldType) {
		    this.fieldName = fieldName;
		    this.fieldType = fieldType;
		}

		@Override
		public int compare(JsonNode a, JsonNode b) {
		    Object valA, valB;
		    Map<String, Object> map_a = new_mapper.convertValue(a, Map.class);
		    Map<String, Object> map_b = new_mapper.convertValue(b, Map.class);
		    valA = map_a.get(fieldName);
		    valB = map_b.get(fieldName);

		    int comp = 0;
		    if (fieldType.equals("long")){
		    	long val_a = Long.parseLong(valA.toString());
		    	long val_b = Long.parseLong(valB.toString());
		    	if (val_a > val_b)
		    		return 1;
		    	else if (val_a < val_b)
		    		return -1;
		    	else
		    		return 0;
		    }
		    return -2;

		}
	}


	public static class PolicyTrigger {//extends PolicyTrigger {

		@NotNull(message="{PolicyTrigger.metricType.NotNull}")
		private String   metricType            = null;

		//the following NotNull does not work as Jackson will alway set 0 if not present
		@NotNull(message="{PolicyTrigger.statWindow.NotNull}")
		//@Min(value=30, message="{PolicyTrigger.statWindow.Min}")
		//@Max(value=1800, message="{PolicyTrigger.statWindow.Max}")
		private int      statWindow = getTriggerDefaultInt("statWindow");

		@NotNull(message="{PolicyTrigger.breachDuration.NotNull}")
		//@Min(value=30, message="{PolicyTrigger.breachDuration.Min}")
		//@Max(value=36000, message="{PolicyTrigger.breachDuration.Max}")
		private int      breachDuration = getTriggerDefaultInt("breachDuration");

		@NotNull(message="{PolicyTrigger.lowerThreshold.NotNull}")
		//@Min(value=1, message="{PolicyTrigger.lowerThreshold.Min}")
		//@Max(value=99, message="{PolicyTrigger.lowerThreshold.Max}")
		private int      lowerThreshold = -1;//getTriggerDefaultInt("lowerThreshold");
		private boolean lowerThreshold_set = false;

		@NotNull(message="{PolicyTrigger.upperThreshold.NotNull}")
		//@Min(value=1, message="{PolicyTrigger.upperThreshold.Min}")
		//@Max(value=99, message="{PolicyTrigger.upperThreshold.Max}")
		private int      upperThreshold = -1;//getTriggerDefaultInt("upperThreshold");
		private boolean upperThreshold_set = false;

		@NotNull(message="{PolicyTrigger.instanceStepCountDown.NotNull}")
		private int      instanceStepCountDown = getTriggerDefaultInt("instanceStepCountDown");

		@NotNull(message="{PolicyTrigger.instanceStepCountUp.NotNull}")
		private int      instanceStepCountUp = getTriggerDefaultInt("instanceStepCountUp");

		@NotNull(message="{PolicyTrigger.stepDownCoolDownSecs.NotNull}")
		//@Min(value=30, message="{PolicyTrigger.stepDownCoolDownSecs.Min}")
		//@Max(value=3600, message="{PolicyTrigger.stepDownCoolDownSecs.Max}")
		private int      stepDownCoolDownSecs = getTriggerDefaultInt("stepDownCoolDownSecs");

		@NotNull(message="{PolicyTrigger.stepUpCoolDownSecs.NotNull}")
		//@Min(value=30, message="{PolicyTrigger.stepUpCoolDownSecs.Min}")
		//@Max(value=3600, message="{PolicyTrigger.stepUpCoolDownSecs.Max}")
		private int      stepUpCoolDownSecs    = getTriggerDefaultInt("stepUpCoolDownSecs");

		public int getTriggerDefaultInt(String key) {
			return Constants.getTriggerDefaultInt(key);
		}

		@AssertTrue(message="{PolicyTrigger.ismetricTypeValid.AssertTrue}")
		private boolean ismetricTypeValid() {
		    String[] metrictype = Constants.metrictype;
		    List<String> metrictypelist = Arrays.asList(metrictype);
		    Set<String> metrictypeset = new HashSet<String>(metrictypelist);
		    return metrictypeset.contains(this.metricType);
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
			this.lowerThreshold_set = true;
			this.lowerThreshold = lowerThreshold;
		}

		public int getUpperThreshold() {
			return this.upperThreshold;
		}

		public void setUpperThreshold(int upperThreshold) {
			this.upperThreshold_set = true;
			this.upperThreshold = upperThreshold;
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

	
	public static class recurringSchedule{

		@NotNull(message="{recurringSchedule.minInstCount.NotNull}")
		@Min(value=1, message="{recurringSchedule.minInstCount.Min}")
		private int minInstCount;

		private int maxInstCount = -1;//if maxInstCount is not appears in the parameter, it's set to 0 which automatically fulfill the validation requirement, actual value should be set after validation

		private boolean maxInstCount_set = false; //true if maxInstCount is set in json or set false;
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

	public static class specificDate{

		@NotNull(message="{specificDate.minInstCount.NotNull}")
		@Min(value=1, message="{specificDate.minInstCount.Min}")
		private int minInstCount;

		private int maxInstCount = -1; //if maxInstCount is not appears in the parameter, it's set to 0 which automatically fulfill the validation requirement, actual value should be set after validation

		private boolean maxInstCount_set = false; //true if maxInstCount is set in json or set false;
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
				//return !(dateformat.parse(firstDateTime).after(dateformat.parse(secondDateTime)));
 	        	return first.before(second);
			} catch (Exception e) {
				logger.info(e.getMessage());
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

	public static class Schedule {
		@NotNull(message="{Policy.timezone.NotNull}") //debug
		private String timezone;

		@Valid
		private List<recurringSchedule> recurringSchedule;

		@Valid
		private List<specificDate> specificDate;

		@AssertTrue(message="{Schedule.isTimeZoneValid.AssertTrue}") //debug
		private boolean isTimeZoneValid() {
			if ((null != this.timezone) && (this.timezone.trim().length() != 0)) {
				logger.debug("In schedules timezone is " + this.timezone);//debug
				Set<String> timezoneset = new HashSet<String>(Arrays.asList(Constants.timezones));
			    if(timezoneset.contains(this.timezone))
			    	return true;
			    else
			    	return false;
			}
			else { //timezone does not exist
				logger.debug("timezone is empty in schedules");//debug
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

	public static class Policy {

		private int      instanceMinCount;

		private int      instanceMaxCount;

		private String timezone;

		private List<PolicyTrigger> policyTriggers;//hold the trigger without/with unknown metricType

		private List<recurringSchedule> recurringSchedule;

		private List<specificDate> specificDate;

		private Schedule schedules;


		public int getInstanceMinCount() {
			return instanceMinCount;
		}

		public void setinstanceMinCount(int instanceMinCount) {
			this.instanceMinCount = instanceMinCount;
		}


		public int getInstanceMaxCount() {
			return instanceMaxCount;
		}

		public void setInstanceMaxCount(int instanceMaxCount) {
			this.instanceMaxCount = instanceMaxCount;
		}

		public String getTimezone() {
			return this.timezone;
		}

		public void setTimezone(String timezone) {
			this.timezone = timezone;
		}

        public List<PolicyTrigger> getPolicyTriggers() {
        	return this.policyTriggers;
        }

        public void setPolicyTriggers(List<PolicyTrigger> policyTriggers) {
        	this.policyTriggers = policyTriggers;
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

		public Schedule getSchedules() {
			return this.schedules;
		}

		public void setSchedules(Schedule schedules) {
			this.schedules = schedules;
		}

	}

	public static class TransferedPolicy {
		@NotNull(message="{Policy.instanceMinCount.NotNull}")
		@Min(value=1, message="{Policy.instanceMinCount.Min}")
		private int      instanceMinCount;

		@NotNull(message="{Policy.instanceMaxCount.NotNull}")
		@Min(value=1, message="{Policy.instanceMaxCount.Min}")
		private int      instanceMaxCount;

		private String timezone;

		@NotNull(message="{Policy.policyTriggers.NotNull}")
		@Valid
		private List<PolicyTrigger> policyTriggers;

		@Valid
		private List<recurringSchedule> recurringSchedule;

		@Valid
		private List<specificDate> specificDate;

		@Valid
		private Schedule schedules;

		private String appType; // for Bean Validation only

/*		@AssertTrue(message="{Policy.instanceMinCount.NotNull}")
		private boolean isInstanceMinCountNotNull(){
			if (this.instanceMinCount == null)
				return false;
			else
				return true;
		}*/

		@AssertTrue(message="{Policy.policyTriggers.NotNull}")
		private boolean isPolicyTriggersNotNull() {
			if (this.policyTriggers == null)
				return false;
			return true;
		}

		@AssertTrue(message="{Policy.isTimeZoneValid.AssertTrue}")
		private boolean isTimeZoneValid() {
			if ((null != this.timezone) && (this.timezone.trim().length() != 0)) {
				logger.info("timezone is not empty: " + this.timezone); //debug
				Set<String> timezoneset = new HashSet<String>(Arrays.asList(Constants.timezones));
			    if(timezoneset.contains(this.timezone))
			    	return true;
			    else
			    	return false;
			}
			else { //timezone does not exist
				logger.info("timezone is empty"); //debug
                    if (((null != this.recurringSchedule) && (this.recurringSchedule.size() > 0)) || ((null != this.specificDate) && (this.specificDate.size() > 0 )) ){
                        return false; //timezone must be specified when we have calendar-based scaling settings
                    }
            }
		    return true;
		}

		@AssertTrue(message="{Policy.isInstanceCountValid.AssertTrue}")
		private boolean isInstanceCountValid() {
		    return this.instanceMinCount <= this.instanceMaxCount;
		}

		//@AssertTrue(message="{Policy.isRecurringScheduleMinInstanceCountValid.AssertTrue}")
		private boolean isRecurringScheduleMinInstanceCountValid() {
			if  (recurringSchedule != null ) {
				for(recurringSchedule rs : recurringSchedule)
					if (rs.minInstCount < this.instanceMinCount)
						return false;
			}
			if ((schedules != null) && (schedules.recurringSchedule != null)) { //need not to check size as foreach loop can handle this
				for(recurringSchedule rs : schedules.recurringSchedule)
					if (rs.minInstCount < this.instanceMinCount)
						return false;
			}
			return true;
		}

		//@AssertTrue(message="{Policy.isSpecificDateMinInstanceCountValid.AssertTrue}")
		private boolean isSpecificDateMinInstanceCountValid() {
			if  (specificDate != null) {
				for(specificDate sd : specificDate)
					if (sd.minInstCount < this.instanceMinCount)
						return false;
			}
			if ((schedules != null) && (schedules.specificDate != null)) { //need not to check size as the foreach loop can handle this
				for(specificDate sd : schedules.specificDate)
					if (sd.minInstCount < this.instanceMinCount)
						return false;
			}
			return true;
		}

		//@AssertTrue(message="{Policy.isRecurringScheduleMaxInstanceCountValid.AssertTrue}")
		private boolean isRecurringScheduleMaxInstanceCountValid() {
			if (recurringSchedule != null ) {
				for(recurringSchedule rs : recurringSchedule)
					if (rs.maxInstCount > this.instanceMaxCount)
						return false;
			}
			if ((schedules != null) && (schedules.recurringSchedule != null)) { // need not to check size as the foreach loop can handle this
				for(recurringSchedule rs : schedules.recurringSchedule)
					if (rs.maxInstCount > this.instanceMaxCount)
						return false;
			}
			return true;
		}

		//@AssertTrue(message="{Policy.isSpecificDateMaxInstanceCountValid.AssertTrue}")
		private boolean isSpecificDateMaxInstanceCountValid() {
			if (specificDate != null )  {
				for(specificDate sd : specificDate)
					if (sd.maxInstCount > this.instanceMaxCount)
						return false;
			}
			if ((schedules != null) && (schedules.specificDate != null)) { //need not to check size as the foreach loop can handle this
				for(specificDate sd : schedules.specificDate)
					if (sd.maxInstCount > this.instanceMaxCount)
						return false;
			}
			return true;
		}

		// with new structure of Policy we define like this, we can not check duplicated metricType trigger, as the late will overwrite previous one
		//@AssertTrue(message="{Policy.isMetricTypeValid.AssertTrue}")
		@AssertTrue(message="{PolicyTrigger.ismetricTypeValid.AssertTrue}")
		private boolean isMetricTypeValid() {
			for (PolicyTrigger trigger : this.policyTriggers){
				if ((trigger.metricType == null) || (!(Arrays.asList(Constants.metrictype).contains(trigger.metricType))))
				    return false;
			}
			return true;
		}

		@AssertTrue(message="{Policy.isMetricTypeValid.AssertTrue}")
		private boolean isMetricTypeUnique() {
			Set<String> metric_types = new HashSet<String>();
			for (PolicyTrigger trigger : this.policyTriggers){
				metric_types.add(trigger.metricType);
			}
			if (metric_types.size() != this.policyTriggers.size())
				return false;
			else 
				return true;
		}

		@AssertTrue(message="{PolicyTrigger.isInstanceStepCountUpValid.AssertTrue}")
		private boolean isInstanceStepCountUpValid() {		//debug we support count instead of percent change only currently
			for (PolicyTrigger trigger : this.policyTriggers){
				if (trigger.instanceStepCountUp > (instanceMaxCount -1))
					return false;
			}
			return true;
		}

		@AssertTrue(message="{PolicyTrigger.isInstanceStepCountDownValid.AssertTrue}")
		private boolean isInstanceStepCountDownValid() {		//debug we support count instead of percent change only currently
			for (PolicyTrigger trigger : this.policyTriggers){
				if (trigger.instanceStepCountDown > instanceMaxCount)
					return false;
			}
			return true;
		}

		@AssertTrue(message="{Policy.isMetricTypeMatched.AssertTrue}")
		private boolean isMetricTypeSupported() {
			String [] supported_metrics = Constants.getMetricTypeByAppType(this.appType);
			logger.info("supported metrics are: " + Arrays.toString(supported_metrics));
			if (supported_metrics != null) {
				for (PolicyTrigger trigger : this.policyTriggers) {
					if  (!(Arrays.asList(supported_metrics).contains(trigger.metricType)))
						return false;
				}
			}
			return true;
		}

		@AssertTrue(message="{Policy.isRecurringScheduleTimeValid.AssertTrue}")
		private boolean isRecurringScheduleTimeValid() {
			if (RecurringScheduleTimeValid(this.recurringSchedule) == false) {
				return false;
			}
			if ((this.schedules != null) && (RecurringScheduleTimeValid(this.schedules.recurringSchedule) == false)) {
				return false;
			}
			return true;

		}
		private boolean RecurringScheduleTimeValid (List<recurringSchedule> recurringSchedule ) {
			if ( recurringSchedule == null )
                return true;
			List<Map<String, String>> recurringScheduleList = new ArrayList<Map<String, String>>();
			for(recurringSchedule rs : recurringSchedule){
				Map<String, String> map = Obj2Map(rs);
				recurringScheduleList.add(map);
			}
			//sort based on startTime
			Collections.sort(recurringScheduleList, new MapComparator1("startTime", "Time"));

			for(int index=0; index <recurringScheduleList.size()-1; index++ ){
				Map<String, String> now = recurringScheduleList.get(index);
				Map<String, String> next = recurringScheduleList.get(index+1);
				 if (now.get("startTime").equals(next.get("startTime"))){ //startTime overlap, so check if repeatOn overlap
					 String [] now_repeatOn_values = now.get("repeatOn").replace("\"", "").replace("[", "").replace("]", "").split(",");
		    		 String [] next_repeatOn_values = next.get("repeatOn").replace("\"", "").replace("[", "").replace("]", "").split(",");
		    		 Set<String> next_set = new HashSet(Arrays.asList(next_repeatOn_values));
		    		 for (String day : now_repeatOn_values) {
		    			 if (next_set.contains(day)){
		    				 return false;
		    			 }
		    		 }
				 }
				 else { //now startTime is earlier than next startTime, then check time range overlap, i.e, if endTime of now if latter than starTime of next
					 try {
						 SimpleDateFormat parser = new SimpleDateFormat("HH:mm");
						 parser.setLenient(false);
						 ParsePosition position = new ParsePosition(0);
						 Date next_start_time = parser.parse(next.get("startTime"), position);
		 	        	 if (( next_start_time == null) || (position.getIndex() != next.get("startTime").length()))
		 	        		 return false;
		 	        	 position = new ParsePosition(0);
					 	 Date now_end_time = parser.parse(now.get("endTime"), position);
		 	        	 if (( now_end_time == null) || (position.getIndex() != now.get("endTime").length()))
		 	        		 return false;
						 if(!(now_end_time.before(next_start_time))) { //time range overlap, now needs to check repeatOn to see if overlap exists
							 String [] now_repeatOn_values = now.get("repeatOn").replace("\"", "").replace("[", "").replace("]", "").split(",");
				    		 String [] next_repeatOn_values = next.get("repeatOn").replace("\"", "").replace("[", "").replace("]", "").split(",");
				    		 Set<String> next_set = new HashSet(Arrays.asList(next_repeatOn_values));
				    		 for (String day : now_repeatOn_values) {
				    			 if (next_set.contains(day)){
				    				 return false;
				    			 }
				    		 }
						 }
					 } catch (Exception e){
						 return false;
					 }
				 }
			}
			return true;
		}

		@AssertTrue(message="{Policy.isSpecificDateTimeValid.AssertTrue}")
		private boolean isSpecificDateTimeValid() {
			if (SpecificDateTimeValid(this.specificDate) == false) {
				return false;
			}
			if ((this.schedules != null) && (SpecificDateTimeValid(this.schedules.specificDate) == false)) {
				return false;
			}
			return true;
		}
		private boolean SpecificDateTimeValid (List<specificDate> specificDate) {
			if (specificDate == null)
                return true;
			List<Map<String, String>> specificDateList = new ArrayList<Map<String, String>>();
			for(specificDate sd : specificDate){
				Map<String, String> map = Obj2Map(sd);
				specificDateList.add(map);
			}
			//sort based on startDateTime
			Collections.sort(specificDateList, new MapComparator2("startDate", "startTime", "DateTime"));

			for(int index=0; index <specificDateList.size()-1; index++ ){
				Map<String, String> now = specificDateList.get(index);
				Map<String, String> next = specificDateList.get(index+1);
				 if ((now.get("startTime").equals(next.get("startTime"))) && (now.get("startDate").equals(next.get("startDate")))){ //startDateTime equal, then must be overlapped
			         return false;
				 }
				 else { //now startTime is earlier than next startTime, then check time range overlap, i.e, if endTime of now if latter than starTime of next
					 try {
						 SimpleDateFormat parser = new SimpleDateFormat("yyyy-MM-dd HH:mm");
						 parser.setLenient(false);
						 String startDateTime = next.get("startDate") + " " + next.get("startTime");
						 String endDatetime = now.get("endDate") + " " + now.get("endTime");
						 ParsePosition position = new ParsePosition(0);
						 Date next_start_time = parser.parse(startDateTime, position);
		 	        	 if (( next_start_time == null) || (position.getIndex() != startDateTime.length()))
		 	        		 return false;
		 	        	 position = new ParsePosition(0);
					 	 Date now_end_time = parser.parse(endDatetime, position);
		 	        	 if (( now_end_time == null) || (position.getIndex() != endDatetime.length()))
		 	        		 return false;
						 if(!(now_end_time.before(next_start_time))) {
				    				 return false;
						 }
					 } catch (Exception e){
						 return false;
					 }
				 }
			}
			return true;
		}

		@AssertTrue(message="{Policy.isScheduleValid.AssertTrue}") //debug
		private boolean isScheduleValid() {
			boolean new_style = false;
			boolean old_style = false;
			if (((this.recurringSchedule != null) && this.recurringSchedule.size() > 0) || ((this.specificDate != null) && (this.specificDate.size() > 0)) || (this.timezone != null)) {
				old_style = true;
			}
			if (this.schedules != null) {
				new_style = true;
			}
			if (new_style && old_style)
				return false;
			return true;
		}

		public int getInstanceMinCount() {
			return instanceMinCount;
		}

		public void setinstanceMinCount(int instanceMinCount) {
			this.instanceMinCount = instanceMinCount;
		}

		/*		@JsonCreator
		public void Policy(@JsonProperty(value="instanceMinCount", required=true) int instanceMinCount) {
			this.instanceMinCount = instanceMinCount;
		}*/

		public int getInstanceMaxCount() {
			return instanceMaxCount;
		}

		public void setInstanceMaxCount(int instanceMaxCount) {
			this.instanceMaxCount = instanceMaxCount;
		}

		public String getTimezone() {
			return this.timezone;
		}

		public void setTimezone(String timezone) {
			this.timezone = timezone;
		}


        public List<PolicyTrigger> getPolicyTriggers() {
        	return this.policyTriggers;
        }

        public void setPolicyTriggers(List<PolicyTrigger> policyTriggers) {
        	this.policyTriggers = policyTriggers;
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

		public Schedule getSchedules() {
			return this.schedules;
		}

		public void setSchedules(Schedule schedules) {
			this.schedules = schedules;
		}

		public String getAppType() {
			return this.appType;
		}

		public void setAppType(String appType) {
			this.appType = appType;
		}
		//this should never be called before validation
		public TransferedPolicy setMaxInstCount() {
			if ( this.recurringSchedule != null ){
				for(recurringSchedule rs : this.recurringSchedule){
					//if(rs.maxInstCount == -1)  //after validation if it's still -1, it only means it's not set in the original json
					if(rs.maxInstCount_set == false) //we use this mark to test if maxInstCount specified in the JSON
						rs.maxInstCount = this.instanceMaxCount;
				}
			}
			if ((this.schedules != null) && (this.schedules.recurringSchedule != null)) { //need not to check size as foreach loop can handle this
				for(recurringSchedule rs : this.schedules.recurringSchedule){
					//if(rs.maxInstCount == -1)  //after validation if it's still -1, it only means it's not set in the original json
					if(rs.maxInstCount_set == false) //we use this mark to test if maxInstCount specified in the JSON
						rs.maxInstCount = this.instanceMaxCount;
				}
			}
			if ( this.specificDate != null ){
				for(specificDate sd : this.specificDate){
					//if(sd.maxInstCount == -1)  //after validation if it's still -1, it only means it's not set in the original json
					if(sd.maxInstCount_set == false) //we use this mark to test if maxInstCount specified in the JSON
						sd.maxInstCount = this.instanceMaxCount;
				}
			}
			if ((this.schedules != null) && (this.schedules.specificDate != null)) { //need not to check size as foreach loop can handle this
				for(specificDate sd : this.schedules.specificDate){
					//if(sd.maxInstCount == -1)  //after validation if it's still -1, it only means it's not set in the original json
					if(sd.maxInstCount_set == false) //we use this mark to test if maxInstCount specified in the JSON
						sd.maxInstCount = this.instanceMaxCount;
				}
			}
			//return  new_mapper.writeValueAsString(this);
			return this;
		}


		public TransferedPolicy transformSchedules() {
			if ((this.recurringSchedule != null) && (this.recurringSchedule.size() > 0)) {
				if (this.schedules == null){
					this.schedules = new Schedule();
				}
				/*
				this.schedules.recurringSchedule = new ArrayList<recurringSchedule>();
				for(recurringSchedule rs : this.recurringSchedule){
					this.schedules.recurringSchedule.add(rs);
				}*/
				this.schedules.setRecurringSchedule(this.recurringSchedule);
				//this.schedules.recurringSchedule = this.recurringSchedule;
			}
			if ((this.specificDate != null) && (this.specificDate.size() >0)) {
				if (this.schedules == null)
					this.schedules = new Schedule();
				this.schedules.setSpecificDate(this.specificDate);
				/*
				this.schedules.specificDate = new ArrayList<specificDate>();
				for(specificDate sd : this.specificDate){
					this.schedules.specificDate.add(sd);
				}*/
				//this.schedules.specificDate = this.specificDate;
			}
			if ((null != this.timezone) && (this.timezone.trim().length() != 0)) {
				if (this.schedules == null)
					this.schedules = new Schedule();
				this.schedules.setTimezone(this.timezone);
			}
			return this;
		}

		public String transformInput() throws JsonParseException, JsonMappingException, IOException{
			if (this.schedules != null) {   //debug need to check output json to see if null field are not encoded
				this.timezone = this.schedules.timezone;
				this.recurringSchedule = this.schedules.recurringSchedule;  //debug why we don't need to call setter here while transformSchedules() can not
				this.specificDate = this.schedules.specificDate;
				this.schedules = null;
			}
			return  new_mapper.writeValueAsString(this);

		}

		public String transformOutput(Map<String, String> supplyment, Map<String, String> service_info) throws JsonParseException, JsonMappingException, IOException{

			String current_json =  new_mapper.writeValueAsString(this);
			current_json = TransferedPolicy.transfer_back(current_json, service_info);
			JsonNode top = new_mapper.readTree(current_json);
			for(String key : supplyment.keySet()) {
				((ObjectNode)top).put(key, supplyment.get(key));
			}


			boolean has_recurringSchedule = false;
			boolean has_specificDate = false;
			boolean has_timezone = false;

			Iterator<Entry<String, JsonNode>> topNodes= top.fields();
		    while(topNodes.hasNext()){
				Entry<String, JsonNode> topNode = topNodes.next();
				String nodestring = topNode.getKey();
				JsonNode subNode = topNode.getValue();

				//JsonNodeFactory nodeFactory = JsonNodeFactory.instance;
				//ObjectNode schedulesnode = nodeFactory.objectNode();

				if(nodestring.equals("recurringSchedule")){
					has_recurringSchedule = true;
					//schedulesnode.put("recurringSchedule", subNode);
				}else if (nodestring.equals("specificDate")){
					//schedulesnode.put("specificDate", subNode);
					has_specificDate = true;
				}else if (nodestring.equals("timezone")){
					//schedulesnode.put("timezone", subNode);
					has_timezone = true;
				}else if (nodestring.equals("policyTriggers")){
					for(JsonNode triggerNode : subNode) {
						Iterator<Entry<String, JsonNode>> triggerNode_items= triggerNode.fields();
						while (triggerNode_items.hasNext()){
							Entry<String, JsonNode> triggerNode_item = triggerNode_items.next();
							String item_name = triggerNode_item.getKey();
							logger.info("item_name: " + item_name);
							JsonNode item_value = triggerNode_item.getValue();
							if(item_name.equals("instanceStepCountDown") && (item_value.asInt()< 0)) {
								((ObjectNode)triggerNode).put("instanceStepCountDown", item_value.asInt()*(-1));
							}
						}
					}
				}
		    }

			//if (schedulesnode.size() > 0)
			//	((ObjectNode)top).put("schedules", schedulesnode);
		    if (has_recurringSchedule)
				((ObjectNode)top).remove("recurringSchedule"); //it should be already in schedules
		    if (has_specificDate)
				((ObjectNode)top).remove("specificDate"); //it should be already in schedules
		    if (has_timezone)
				((ObjectNode)top).remove("timezone"); //it should be already in schedules
			//}

		    return top.toString();


/*			Map<String, String> obj_map = Obj2Map(this);
			Map<String, String> new_map = new HashMap<String, String>();
			List<Map<String, String>> recurringScheduleList = new ArrayList<Map<String, String>>();
			List<Map<String, String>> specificDateList = new ArrayList<Map<String, String>>();
			for (String key : obj_map.keySet()) {
				if (!(key.equals("recurringSchedule")) && !(key.equals("specificDate"))) {
					new_map.put(key, obj_map.get(key));
				}
			}
			for(recurringSchedule rs : this.recurringSchedule){
				Map<String, String> rs_map = Obj2Map(rs);
				rs_map.put("type", "RECURRING");
				recurringScheduleList.add(rs_map);
			}
			for(specificDate sd : this.specificDate){
				Map<String, String> sd_map = Obj2Map(sd);
				sd_map.put("type", "SPECIALDATE");
				specificDateList.add(sd_map);
			}
			new_map.put("recurringSchedule", recurringScheduleList.toString());
			new_map.put("specificDate", specificDateList.toString());
			for(String key : supplyment.keySet()) {
				new_map.put(key, supplyment.get(key));
			}
			return  new_mapper.writeValueAsString(new_map); */


		}


		public static String transfer(String current_json, Map<String, String> service_info) throws JsonParseException, JsonMappingException, IOException {
			String new_string = current_json;

			JsonNode top = new_mapper.readTree(current_json);
			JsonNode new_top = new_mapper.readTree(new_string);

			String appType = service_info.get("appType");
			if (appType != null) {
				((ObjectNode)new_top).put("appType", appType);
			}
			//return new_top.toString();
			return new_mapper.writeValueAsString(new_top);
		}

		public static String transfer_back(String current_json, Map<String, String> service_info) throws JsonParseException, JsonMappingException, IOException {
			String new_string = current_json;

			JsonNode top = new_mapper.readTree(current_json);
			JsonNode new_top = new_mapper.readTree(new_string);

			((ObjectNode)new_top).remove("appType");
			//return new_top.toString();
			return new_mapper.writeValueAsString(new_top);
		}

	}



	public static class PolicyEnbale {
		@NotNull(message="{PolicyEnbale.enable.NotNull}")
		private boolean enable;

		public boolean getEable() {
			return this.enable;
		}

		public void setEnable(boolean enable) {
			this.enable = enable;
		}
		public String transformInput() throws JsonParseException, JsonMappingException, IOException{
			Map<String, String> result = new HashMap<String, String>();
			if (this.enable == true) {
				result.put("state", "enabled");
			}
			else {
				result.put("state", "disabled");
			}
			return  new_mapper.writeValueAsString(result);
		}
	}

	public static class ScalingTrigger {
		@NotNull( message="{ScalingTrigger.metrics.NotNull}")
		private String metrics;

		@NotNull( message="{ScalingTrigger.threshold.NotNull}")
		private int threshold;

		@NotNull( message="{ScalingTrigger.thresholdType.NotNull}")
		private String thresholdType;

		@NotNull( message="{ScalingTrigger.breachDuration.NotNull}")
		private int breachDuration;

		@NotNull( message="{ScalingTrigger.triggerType.NotNull}")
		@Min(value=0, message="{ScalingTrigger.triggerType.Min}")
		@Max(value=1, message="{ScalingTrigger.triggerType.Max}")
		private int triggerType;


		public String getMetrics() {
			return this.metrics;
		}

		public void setMetrics(String metrics){
			this.metrics = metrics;
		}

		public int getThreshold() {
			return this.threshold;
		}

		public void setThreshold(int threshold) {
			this.threshold = threshold;
		}

		public String getThresholdType() {
			return this.thresholdType;
		}

		public void setThresholdType(String thresholdType) {
			this.thresholdType = thresholdType;
		}

		public int getBreachDuration() {
			return this.breachDuration;
		}

		public void setBreachDuration(int breachDuration) {
			this.breachDuration = breachDuration;
		}

		public int getTriggerType() {
			return this.triggerType;
		}

		public void setTriggerType(int triggerType) {
			this.triggerType = triggerType;
		}
	}

	public static class HistoryData {
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
		//private List<ScalingTrigger> trigger;
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

	public static class ScalingHistory {
		@Valid
		private List<HistoryData> data;

		public List<HistoryData> getData() {
			return this.data;
		}

		public void setData(List<HistoryData> data) {
			this.data = data;
		}

	}

	public static class Metric {
		@NotNull(message="{Metric.name.NotNull}")
	    private String name;

		@NotNull(message="{Metric.value.NotNull}")
	    private String value;

		@NotNull(message="{Metric.category.NotNull}")
	    private String category;

		@NotNull(message="{Metric.group.NotNull}")
	    private String group;

		@NotNull(message="{Metric.timestamp.NotNull}")
	    private long timestamp;

		@NotNull(message="{Metric.unit.NotNull}")
	    private String unit;

	    private String desc="";

	    public String getName() {
	    	return this.name;
	    }

	    public void setName(String name) {
	    	this.name = name;
	    }

	    public String getValue() {
	    	return this.value;
	    }

	    public void setValue(String value) {
	    	this.value = value;
	    }

	    public String getCategory() {
	    	return this.category;
	    }

	    public void setCategory(String category) {
	    	this.category = category;
	    }

	    public String getGroup() {
	    	return this.group;
	    }

	    public void setGroup(String group) {
	    	this.group = group;
	    }

	    public long getTimestamp() {
	    	return this.timestamp;
	    }

	    public void setTimestamp(long timestamp) {
	    	this.timestamp = timestamp;
	    }

	    public String getUnit() {
	    	return this.unit;
	    }

	    public void setUnit(String unit) {
	    	this.unit = unit;
	    }

        public String getDesc() {
        	return this.desc;
        }

        public void setDesc(String desc) {
        	this.desc = desc;
        }
	}

	public static class InstanceMetric {
		@NotNull( message="{InstanceMetric.instanceIndex.NotNull}")
	    private int instanceIndex;

		@NotNull( message="{InstanceMetric.timestamp.NotNull}")
		private long timestamp;

		@NotNull( message="{InstanceMetric.instanceId.NotNull}")
	    private String instanceId;

		@NotNull( message="{InstanceMetric.metrics.NotNull}")
		@Valid
	    private List<Metric> metrics = new LinkedList<Metric>();

	    public int getInstanceIndex() {
	    	return this.instanceIndex;
	    }

	    public void setInstanceIndex(int instanceIndex) {
	    	this.instanceIndex = instanceIndex;
	    }

		public long getTimestamp() {
			return this.timestamp;
		}

		public void setTimestamp(long timestamp) {
			this.timestamp = timestamp;
		}

	    public String getInstanceId() {
	    	return this.instanceId;
	    }

	    public void setInstanceId(String instanceId) {
	    	this.instanceId = instanceId;
	    }

	    public List<Metric> getMetrics() {
	    	return this.metrics;
	    }

	    public void setMetrics(List<Metric> metrics) {
	    	this.metrics = metrics;
	    }
	}
	public static class MetricData {
		@NotNull(message="{MetricData.appId.NotNull}")
		private String appId;

		@NotNull(message="{MetricData.appName.NotNull}")
		private String appName;

		@NotNull(message="{MetricData.appType.NotNull}")
		private String appType;

		@NotNull(message="{MetricData.timestamp.NotNull}")
	    private long timestamp;

		@Valid
		private List<InstanceMetric> instanceMetrics;

		public String getAppId(){
			return this.appId;
		}

		public void setAppId(String appId){
			this.appId = appId;
		}

		public String getAppName() {
			return this.appName;
		}

		public void setAppName(String appName) {
			this.appName = appName;
		}

		public String getAppType() {
			return this.appType;
		}

		public void setAppType(String appType) {
			this.appType = appType;
		}

		public List<InstanceMetric> getInstanceMetrics() {
			return this.instanceMetrics;
		}

		public void setInstanceMetrics(List<InstanceMetric> instanceMetrics) {
			this.instanceMetrics = instanceMetrics;
		} 

	    public long getTimestamp() {
	    	return this.timestamp;
	    }

	    public void setTimestamp(long timestamp){
	    	this.timestamp = timestamp;
	    }

	}

	public static class Metrics {
		@Valid
		private List<MetricData> data;

		public List<MetricData> getData() {
			return this.data;
		}

		public void setData(List<MetricData> data) {
			this.data = data;
		}

		public String transformOutput() throws JsonParseException, JsonMappingException, IOException{

/*			List<Map<String, String>> metricdatalist  = new ArrayList<Map<String, String>>();
			for (MetricData metricdata : this.data){
				Map<String, String> data_map = Obj2Map(metricdata);
				metricdatalist.add(data_map);
			}

			Collections.sort(metricdatalist, new MapComparator1("timestamp", "long"));

			List<Map<String, String>> sub_list;
		    long last_timestamp;
		    int max_len = Integer.parseInt(ConfigManager.get("maxMetricRecord"));
		    logger.info("Current maxMetricRecord returned is " + max_len + " and current number of metric record is " + metricdatalist.size());
		    if(metricdatalist.size() > max_len){
		    	sub_list = metricdatalist.subList(0, max_len);
		    	last_timestamp = Long.valueOf(metricdatalist.get(max_len-1).get("timestamp"));
		    }
		    else {
		    	sub_list = metricdatalist;
		    	last_timestamp = 0;
		    }
		    ObjectNode jNode = new_mapper.createObjectNode();
		    jNode.put("data", new_mapper.convertValue(sub_list, JsonNode.class));
		    jNode.put("timestamp", last_timestamp);


		    return jNode.asText();*/


			String current_json =  new_mapper.writeValueAsString(this.data);
			JsonNode top = new_mapper.readTree(current_json);
			JsonObjectComparator comparator = new JsonObjectComparator("timestamp", "long");
			Iterator<JsonNode> elements = top.elements();
			List<JsonNode> elements_sorted = new ArrayList<JsonNode>();
			while(elements.hasNext()){
				elements_sorted.add(elements.next());
			}
			Collections.sort(elements_sorted, comparator);

			ObjectNode jNode = new_mapper.createObjectNode();

			List<JsonNode> sub_list;
		    long last_timestamp;
		    int max_len = Integer.parseInt(ConfigManager.get("maxMetricRecord"));
		    logger.info("Current maxMetricRecord returned is " + max_len + " and current number of metric record is " + elements_sorted.size());
		    if(elements_sorted.size() > max_len){
		    	sub_list = elements_sorted.subList(0, max_len);
		    	JsonNode last_metric = elements_sorted.get(max_len-1);
		    	Map<String, Object> last_metric_map = new_mapper.convertValue(last_metric, Map.class);
		    	last_timestamp = Long.valueOf((String)last_metric_map.get("timestamp"));
		    }
		    else {
		    	sub_list = elements_sorted;
		    	last_timestamp = 0;
		    }

		    JsonNodeFactory factory = JsonNodeFactory.instance;
		    ArrayNode aaData = new ArrayNode(factory);
		    aaData.addAll(sub_list);
			jNode.put("data", aaData);
		    jNode.put("timestamp", last_timestamp);
			return jNode.toString();

		}
	}

	public static HibernateValidatorConfiguration getPolicyRange() throws JsonParseException, JsonMappingException, IOException{
		HibernateValidatorConfiguration config = Validation.byProvider( HibernateValidator.class ).configure();

		config = getPolicyTriggerRange(config);
		return config;
	}

	public static HibernateValidatorConfiguration getPolicyTriggerRange(HibernateValidatorConfiguration config) {

		 Map<String, Map<String, Map<String, Map<String, String>>>> mulit_range = Constants.getTriggerRangeByTriggerType();
		 logger.debug("mulit trigger range: " + mulit_range.toString());
		 for (String class_type : mulit_range.keySet()){
			 Map<String, Map<String, Map<String, String>>> range = mulit_range.get(class_type);

			 for (String key : range.keySet()) {
				 Map<String, Map<String, String>> value = range.get(key);
				 for(String value_key : value.keySet()){
					 Map<String, String> value_item = value.get(value_key);
					 ConstraintMapping mapping = new ConstraintMapping();
					 if (value_key.endsWith("Min")){
						 mapping.type(PolicyTrigger.class).property(key, ElementType.FIELD).constraint( new MinDef().value(Integer.parseInt(value_item.get("value"))).message(value_item.get("message")));
					 }
					 else if(value_key.endsWith("Max")){
						 mapping.type(PolicyTrigger.class).property(key, ElementType.FIELD).constraint( new MaxDef().value(Integer.parseInt(value_item.get("value"))).message(value_item.get("message")));
					 }
					 else if(value_key.endsWith("NotNull")){
						 mapping.type(PolicyTrigger.class).property(key, ElementType.FIELD).constraint( new NotNullDef().message(value_item.get("message")));
					 }
					 config.addMapping( mapping );
				 }
			 }
		 }
		 return config;
	}


	public static String transformHistory(List<HistoryData> scalinghistory) throws JsonParseException, JsonMappingException, IOException{
		String current_json =  new_mapper.writeValueAsString(scalinghistory);
		JsonNode top = new_mapper.readTree(current_json);
		Iterator<JsonNode> elements = top.elements();
		int instances= -1, adjustment=-1;
		while (elements.hasNext()){
			JsonNode history_data = elements.next();
			Iterator<Entry<String, JsonNode>> history_data_items= history_data.fields();
			while(history_data_items.hasNext()){
				Entry<String, JsonNode> Node = history_data_items.next();
				String nodestring = Node.getKey();
				JsonNode subNode = Node.getValue();
				if(nodestring.equals("status")){
					switch(subNode.asInt()){
						case 1: ((ObjectNode)history_data).put("status", "READY");
        	                    break;
			        	case 2: ((ObjectNode)history_data).put("status", "REALIZING");
			        	        break;
			        	case 3: ((ObjectNode)history_data).put("status", "COMPLETED");
				                break;
				        case -1: ((ObjectNode)history_data).put("status", "FAILED");
				                  break;
				        default: break;
					}
				} else if (nodestring.equals("trigger")){
					//Iterator<JsonNode> trigger_elements = subNode.elements();
					Iterator<Entry<String, JsonNode>> trigger_items= subNode.fields();
					//while (trigger_elements.hasNext()){
					while (trigger_items.hasNext()){
						//JsonNode trigger_element = trigger_elements.next();
						Entry<String, JsonNode> item_node = trigger_items.next();
						//Iterator<Entry<String, JsonNode>> trigger_items= trigger_element.fields();
						String item_name = item_node.getKey();
						JsonNode item_value = item_node.getValue();
						if(item_name.equals("triggerType")){
							switch(item_value.asInt()){
								case 0: ((ObjectNode)subNode).put("triggerType", "MonitorEvent");
	    	                            break;
					        	case 1: ((ObjectNode)subNode).put("triggerType", "PolicyChange");
					        	        break;
						        default: break;
							}
						}
						/*
						while(trigger_items.hasNext()){
							Entry<String, JsonNode> trigger_item = trigger_items.next();
							String trigger_key = trigger_item.getKey();
							JsonNode trigger_value = trigger_item.getValue();
							if(trigger_key.equals("triggerType")){
								switch(trigger_value.asInt()){
									case 0: ((ObjectNode)trigger_element).put("triggerType", "MonitorEvent");
		    	                            break;
						        	case 1: ((ObjectNode)trigger_element).put("triggerType", "PolicyChange");
						        	        break;
							        default: break;
								}
							}
						}*/
					}
				} else if (nodestring.equals("instances")){
					instances = subNode.asInt();
				} else if (nodestring.equals("adjustment")){
					adjustment = subNode.asInt();
				}
			}
			((ObjectNode)history_data).put("instancesBefore", instances - adjustment);
			((ObjectNode)history_data).put("instancesAfter", instances);
			((ObjectNode)history_data).remove("instances");
			((ObjectNode)history_data).remove("adjustment");
		}


		JsonObjectComparator comparator = new JsonObjectComparator("startTime", "long");
		elements = top.elements();
		List<JsonNode> elements_sorted = new ArrayList<JsonNode>();
		while(elements.hasNext()){
			elements_sorted.add(elements.next());
		}
		Collections.sort(elements_sorted, comparator);

		ObjectNode jNode = new_mapper.createObjectNode();


		List<JsonNode> sub_list;
	    long last_timestamp;
	    int max_len = Integer.parseInt(ConfigManager.get("maxHistoryRecord"));
	    logger.info("Current maxHistoryRecord returned is " + max_len + " and current number of history record is " + elements_sorted.size());
	    if(elements_sorted.size() > max_len){
	    	sub_list = elements_sorted.subList(0, max_len);
	    	JsonNode last_history = elements_sorted.get(max_len-1);
	    	Map<String, Object> last_history_map = new_mapper.convertValue(last_history, Map.class);
	    	last_timestamp = Long.valueOf((String)last_history_map.get("startTime"));
	    }
	    else {
	    	sub_list = elements_sorted;
	    	last_timestamp = 0;
	    }

	    JsonNodeFactory factory = JsonNodeFactory.instance;
	    ArrayNode aaData = new ArrayNode(factory);
	    aaData.addAll(sub_list);
		jNode.put("data", aaData);
	    jNode.put("timestamp", last_timestamp);
		return jNode.toString();

		/*
	    Map<String, Object> new_result = new HashMap<String, Object>();
	    new_result.put("data", sub_list);
	    new_result.put("timestamp", last_timestamp);

		return  new_mapper.writeValueAsString(new_result);
		*/
	}



	public static Map<String, Object> parsePolicy(String jsonString, Map<String, String> service_info, HttpServletRequest httpServletRequest) throws JsonParseException, JsonMappingException, IOException{
		 List<String> violation_message = new ArrayList<String>();
		 Map<String, Object> result = new HashMap<String, Object>();
		 result.put("valid", false);
		 logger.info("received policy : " + jsonString); //debug
		 Policy oldpolicy = new_mapper.readValue(jsonString, Policy.class); //for syntax error check
		 String transfered_json = TransferedPolicy.transfer(jsonString, service_info);
		 logger.info("transfered policy after update with service_information : " + transfered_json); //debug
		 TransferedPolicy policy = new_mapper.readValue(transfered_json, TransferedPolicy.class);
		 logger.info("we get policy as " + (Obj2Map(policy)).toString());//debug
		 //additional data manipulation and check again
		 policy = policy.setMaxInstCount();
		 HibernateValidatorConfiguration config = ProtoType.getPolicyRange();
		 ValidatorFactory vf = config.buildValidatorFactory();
		 //Validator validator = factory.getValidator();

		 //ValidatorFactory vf = Validation.buildDefaultValidatorFactory();
		 //Validator validator = vf.getValidator();
		 Locale locale = LocaleUtil.getLocale(httpServletRequest);
		 MessageInterpolator interpolator = new LocaleSpecificMessageInterpolator(vf.getMessageInterpolator(), locale);
		 Validator validator = vf.usingContext().messageInterpolator(interpolator).getValidator();
		 Set<ConstraintViolation<TransferedPolicy>> set = validator.validate(policy);
		 if (set.size() > 0 ){
			 for (ConstraintViolation<TransferedPolicy> constraintViolation : set) {
				 violation_message.add(constraintViolation.getMessage());
			 }
			 result.put("violation_message", violation_message);
			 return result;
		 }

		 String new_json = policy.transformInput();
		 logger.info("policy before trigger back : " + new_json); //debug
		 new_json = TransferedPolicy.transfer_back(new_json, service_info);
		 result.put("valid", true);
		 logger.info("send out policy: " + new_json); //debug
		 result.put("new_json", new_json);
		 return result;
	}

	public static Map<String, Object> parsePolicyOutput(String jsonString, Map<String, String> supplyment, Map<String, String> service_info, HttpServletRequest httpServletRequest) throws JsonParseException, JsonMappingException, IOException{
		 List<String> violation_message = new ArrayList<String>();
		 Map<String, Object> result = new HashMap<String, Object>();
		 result.put("valid", false);
		 logger.info("received json: " + jsonString); 
		 String transfered_json = TransferedPolicy.transfer(jsonString, service_info);
		 logger.info("transfered policy after update with service_information : " + transfered_json); //debug
		 TransferedPolicy policy = new_mapper.readValue(transfered_json, TransferedPolicy.class);
		 logger.info("we get policy as " + (Obj2Map(policy)).toString());//debug
		 ValidatorFactory vf = Validation.buildDefaultValidatorFactory();
		 //Validator validator = vf.getValidator();
		 //here don't call getPolicyRange and strictly check value range as in parsePolicy
		 Locale locale = LocaleUtil.getLocale(httpServletRequest);
		 MessageInterpolator interpolator = new LocaleSpecificMessageInterpolator(vf.getMessageInterpolator(), locale);
		 Validator validator = vf.usingContext().messageInterpolator(interpolator).getValidator();
		 Set<ConstraintViolation<TransferedPolicy>> set = validator.validate(policy);
		 if (set.size() > 0 ){
			 for (ConstraintViolation<TransferedPolicy> constraintViolation : set) {
				 violation_message.add(constraintViolation.getMessage());
			 }
			 result.put("violation_message", violation_message);
			 return result;
		 }

		 //additional data manipulation
		 policy = policy.transformSchedules();
		 String new_json = policy.transformOutput(supplyment, service_info);
		 result.put("valid", true);
		 result.put("new_json", new_json);
		 return result;
	}

	public static Map<String, Object> parsePolicyEnable(String jsonString, HttpServletRequest httpServletRequest) throws JsonParseException, JsonMappingException, IOException{
		 List<String> violation_message = new ArrayList<String>();
		 Map<String, Object> result = new HashMap<String, Object>();
		 result.put("valid", false);
		 PolicyEnbale policyEnable = new_mapper.readValue(jsonString, PolicyEnbale.class);
		 ValidatorFactory vf = Validation.buildDefaultValidatorFactory();
		 //Validator validator = vf.getValidator();
		 Locale locale = LocaleUtil.getLocale(httpServletRequest);
		 MessageInterpolator interpolator = new LocaleSpecificMessageInterpolator(vf.getMessageInterpolator(), locale);
		 Validator validator = vf.usingContext().messageInterpolator(interpolator).getValidator();
		 Set<ConstraintViolation<PolicyEnbale>> set = validator.validate(policyEnable);
		 if (set.size() > 0 ){
			 for (ConstraintViolation<PolicyEnbale> constraintViolation : set) {
				 violation_message.add(constraintViolation.getMessage());
			 }
			 result.put("violation_message", violation_message);
			 return result;
		 }

		 //additional data manipulation
		 String new_json = policyEnable.transformInput();
		 result.put("valid", true);
		 result.put("new_json", new_json);
		 return result;
	}

	public static JavaType getCollectionType(Class<?> collectionClass, Class<?>... elementClasses) {
	         return new_mapper.getTypeFactory().constructParametricType(collectionClass, elementClasses);
	}

	public static Map<String, Object> parseScalingHistory(String jsonString, HttpServletRequest httpServletRequest) throws JsonParseException, JsonMappingException, IOException{
		 List<String> violation_message = new ArrayList<String>();
		 Map<String, Object> result = new HashMap<String, Object>();
		 result.put("valid", false);
		 JavaType javaType = getCollectionType(ArrayList.class, HistoryData.class);
		 new_mapper.configure(DeserializationFeature.ACCEPT_SINGLE_VALUE_AS_ARRAY, true);
		 List<HistoryData> scalinghistory = (List<HistoryData>)new_mapper.readValue(jsonString, javaType);
		 ValidatorFactory vf = Validation.buildDefaultValidatorFactory();
		 //Validator validator = vf.getValidator();
		 Locale locale = LocaleUtil.getLocale(httpServletRequest);
		 MessageInterpolator interpolator = new LocaleSpecificMessageInterpolator(vf.getMessageInterpolator(), locale);
		 Validator validator = vf.usingContext().messageInterpolator(interpolator).getValidator();
		 Set<ConstraintViolation<List<HistoryData>>> set = validator.validate(scalinghistory);
		 if (set.size() > 0 ){
			 for (ConstraintViolation<List<HistoryData>> constraintViolation : set) {
				 violation_message.add(constraintViolation.getMessage());
			 }
			 result.put("violation_message", violation_message);
			 return result;
		 }

		 //additional data manipulation
     	 String new_json = transformHistory(scalinghistory);
		 result.put("valid", true);
		 result.put("new_json", new_json);
		 return result;
	}

	public static Map<String, Object> parseMetrics(String jsonString, HttpServletRequest httpServletRequest) throws JsonParseException, JsonMappingException, IOException{
		 List<String> violation_message = new ArrayList<String>();
		 Map<String, Object> result = new HashMap<String, Object>();
		 result.put("valid", false);
		 //JavaType javaType = getCollectionType(ArrayList.class, HistoryData.class);
		 //new_mapper.configure(DeserializationFeature.ACCEPT_SINGLE_VALUE_AS_ARRAY, true);
		 logger.info("Received metrics: " + jsonString);
		 Metrics metrics = new_mapper.readValue(jsonString, Metrics.class);
		 
		 ValidatorFactory vf = Validation.buildDefaultValidatorFactory();
		 //Validator validator = vf.getValidator();
		 Locale locale = LocaleUtil.getLocale(httpServletRequest);
		 MessageInterpolator interpolator = new LocaleSpecificMessageInterpolator(vf.getMessageInterpolator(), locale);
		 Validator validator = vf.usingContext().messageInterpolator(interpolator).getValidator();
		 Set<ConstraintViolation<Metrics>> set = validator.validate(metrics);
		 if (set.size() > 0 ){
			 for (ConstraintViolation<Metrics> constraintViolation : set) {
				 violation_message.add(constraintViolation.getMessage());
			 }
			 result.put("violation_message", violation_message);
			 return result;
		 }
          

		 //additional data manipulation
    	 String new_json = metrics.transformOutput();
		 result.put("valid", true);
		 result.put("new_json", new_json);
		 return result;
	}

}
