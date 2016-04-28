package org.cloudfoundry.autoscaler.api.validation;

import java.io.IOException;
import java.text.ParsePosition;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.Date;
import java.util.HashSet;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.Map.Entry;

import javax.validation.Valid;
import javax.validation.constraints.AssertTrue;
import javax.validation.constraints.Min;
import javax.validation.constraints.NotNull;

import org.cloudfoundry.autoscaler.api.validation.BeanValidation.CommonComparator;
import org.cloudfoundry.autoscaler.api.validation.BeanValidation.DateTimeComparator;
import org.cloudfoundry.autoscaler.common.Constants;

import com.fasterxml.jackson.core.JsonParseException;
import com.fasterxml.jackson.databind.JsonMappingException;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.node.ObjectNode;

public class TransferedPolicy {
	@NotNull(message="{Policy.instanceMinCount.NotNull}")
	@Min(value=1, message="{Policy.instanceMinCount.Min}")
	private int      instanceMinCount;

	@NotNull(message="{Policy.instanceMaxCount.NotNull}")
	@Min(value=1, message="{Policy.instanceMaxCount.Min}")
	private int      instanceMaxCount;

	@NotNull(message="{Policy.policyTriggers.NotNull}")
	@Valid
	private List<PolicyTrigger> policyTriggers;

	private String timezone;

	private List<recurringSchedule> recurringSchedule;

	private List<specificDate> specificDate;
	
	@Valid
	private Schedule schedules;

	private String appType; // for Bean Validation only


	@AssertTrue(message="{Policy.policyTriggers.NotNull}")
	private boolean isPolicyTriggersNotNull() {
		if (this.policyTriggers == null)
			return false;
		return true;
	}


	@AssertTrue(message="{Policy.isInstanceCountValid.AssertTrue}")
	private boolean isInstanceCountValid() {
	    return this.instanceMinCount <= this.instanceMaxCount;
	}

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
		BeanValidation.logger.info("supported metrics are: " + Arrays.toString(supported_metrics));
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
			Map<String, String> map = BeanValidation.Obj2Map(rs);
			recurringScheduleList.add(map);
		}
		//sort based on startTime
		Collections.sort(recurringScheduleList, new CommonComparator("startTime", "Time"));

		for(int index=0; index <recurringScheduleList.size()-1; index++ ){
			Map<String, String> now = recurringScheduleList.get(index);
			Map<String, String> next = recurringScheduleList.get(index+1);
			 if (now.get("startTime").equals(next.get("startTime"))){ //startTime overlap, so check if repeatOn overlap
				 String [] now_repeatOn_values = now.get("repeatOn").replace("\"", "").replace("[", "").replace("]", "").split(",");
	    		 String [] next_repeatOn_values = next.get("repeatOn").replace("\"", "").replace("[", "").replace("]", "").split(",");
	    		 Set<String> next_set = new HashSet<String>(Arrays.asList(next_repeatOn_values));
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
			    		 Set<String> next_set = new HashSet<String>(Arrays.asList(next_repeatOn_values));
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
			Map<String, String> map = BeanValidation.Obj2Map(sd);
			specificDateList.add(map);
		}
		//sort based on startDateTime
		Collections.sort(specificDateList, new DateTimeComparator("startDate", "startTime", "DateTime"));

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

	public String getAppType() {
		return this.appType;
	}

	public void setAppType(String appType) {
		this.appType = appType;
	}
	//called after validation
	public TransferedPolicy setMaxInstCount() {
		if ((this.schedules != null) && (this.schedules.recurringSchedule != null)) { 
			for(recurringSchedule rs : this.schedules.recurringSchedule){
				if(rs.maxInstCount_set == false) //maxInstCount is not specified in the JSON
					rs.maxInstCount = this.instanceMaxCount;
			}
			for(specificDate sd : this.schedules.specificDate){
				if(sd.maxInstCount_set == false) //maxInstCount is not specified in the JSON
					sd.maxInstCount = this.instanceMaxCount;
			}
		}
		return this;
	}


	public TransferedPolicy transformSchedules() {
		if ((this.recurringSchedule != null) && (this.recurringSchedule.size() > 0)) {
			if (this.schedules == null){
				this.schedules = new Schedule();
			}
			this.schedules.setRecurringSchedule(this.recurringSchedule);
		}
		if ((this.specificDate != null) && (this.specificDate.size() >0)) {
			if (this.schedules == null)
				this.schedules = new Schedule();
			this.schedules.setSpecificDate(this.specificDate);
		}
		if ((null != this.timezone) && (this.timezone.trim().length() != 0)) {
			if (this.schedules == null)
				this.schedules = new Schedule();
			this.schedules.setTimezone(this.timezone);
		}
		return this;
	}

	public String transformInput() throws JsonParseException, JsonMappingException, IOException{
		if (this.schedules != null) {   
			this.timezone = this.schedules.timezone;
			this.recurringSchedule = this.schedules.recurringSchedule;  
			this.specificDate = this.schedules.specificDate;
			this.schedules = null;
		}
		return  BeanValidation.new_mapper.writeValueAsString(this);

	}

	public String transformOutput(Map<String, String> supplyment, Map<String, String> service_info) throws JsonParseException, JsonMappingException, IOException{

		String current_json =  BeanValidation.new_mapper.writeValueAsString(this);
		current_json = TransferedPolicy.unpackServiceInfo(current_json, service_info);
		JsonNode top = BeanValidation.new_mapper.readTree(current_json);
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

			if(nodestring.equals("recurringSchedule")){
				has_recurringSchedule = true;
			}else if (nodestring.equals("specificDate")){
				has_specificDate = true;
			}else if (nodestring.equals("timezone")){
				has_timezone = true;
			}else if (nodestring.equals("policyTriggers")){
				for(JsonNode triggerNode : subNode) {
					Iterator<Entry<String, JsonNode>> triggerNode_items= triggerNode.fields();
					while (triggerNode_items.hasNext()){
						Entry<String, JsonNode> triggerNode_item = triggerNode_items.next();
						String item_name = triggerNode_item.getKey();
						BeanValidation.logger.info("item_name: " + item_name);
						JsonNode item_value = triggerNode_item.getValue();
						if(item_name.equals("instanceStepCountDown") && (item_value.asInt()< 0)) {
							((ObjectNode)triggerNode).put("instanceStepCountDown", item_value.asInt()*(-1));
						}
					}
				}
			}
	    }

	    if (has_recurringSchedule)
			((ObjectNode)top).remove("recurringSchedule"); 
	    if (has_specificDate)
			((ObjectNode)top).remove("specificDate"); 
	    if (has_timezone)
			((ObjectNode)top).remove("timezone"); 

	    return top.toString();

	}


	public static String packServiceInfo(String current_json, Map<String, String> service_info) throws JsonParseException, JsonMappingException, IOException {

		JsonNode new_top = BeanValidation.new_mapper.readTree(current_json);

		String appType = service_info.get("appType");
		if (appType != null) {
			((ObjectNode)new_top).put("appType", appType);
		}
		return BeanValidation.new_mapper.writeValueAsString(new_top);
	}

	public static String unpackServiceInfo(String current_json, Map<String, String> service_info) throws JsonParseException, JsonMappingException, IOException {

		JsonNode new_top = BeanValidation.new_mapper.readTree(current_json);

		((ObjectNode)new_top).remove("appType");
		return BeanValidation.new_mapper.writeValueAsString(new_top);
	}

}