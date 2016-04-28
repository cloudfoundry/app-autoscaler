package org.cloudfoundry.autoscaler.api.validation;

import java.io.IOException;
import java.lang.annotation.ElementType;
import java.lang.reflect.Method;
import java.text.ParsePosition;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Collections;
import java.util.Comparator;
import java.util.Date;
import java.util.HashMap;
import java.util.Iterator;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Map.Entry;
import java.util.Set;

import javax.servlet.http.HttpServletRequest;
import javax.validation.ConstraintViolation;
import javax.validation.MessageInterpolator;
import javax.validation.Validation;
import javax.validation.Validator;
import javax.validation.ValidatorFactory;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.common.Constants;
import org.cloudfoundry.autoscaler.common.util.ConfigManager;
import org.cloudfoundry.autoscaler.common.util.LocaleSpecificMessageInterpolator;
import org.cloudfoundry.autoscaler.common.util.LocaleUtil;
import org.hibernate.validator.HibernateValidator;
import org.hibernate.validator.HibernateValidatorConfiguration;


import org.hibernate.validator.cfg.ConstraintMapping;
import org.hibernate.validator.cfg.defs.MaxDef;
import org.hibernate.validator.cfg.defs.MinDef;
import org.hibernate.validator.cfg.defs.NotNullDef;

import com.fasterxml.jackson.core.JsonParseException;
import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.JavaType;
import com.fasterxml.jackson.databind.JsonMappingException;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.node.ArrayNode;
import com.fasterxml.jackson.databind.node.JsonNodeFactory;
import com.fasterxml.jackson.databind.node.ObjectNode;

public class BeanValidation {
	private static final String CLASS_NAME = BeanValidation.class.getName();
	static final Logger logger     = Logger.getLogger(CLASS_NAME);
	static final ObjectMapper new_mapper = new ObjectMapper().configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false)
			                                                         .configure(DeserializationFeature.ACCEPT_FLOAT_AS_INT, false)
			                                                         .configure(DeserializationFeature.FAIL_ON_NULL_FOR_PRIMITIVES, true);
			                                                        // .configure(SerializationFeature.INDENT_OUTPUT, true);
			                                                        // .configure(DeserializationFeature.FAIL_ON_MISSING_CREATOR_PROPERTIES, true);

	public static Map<String, String> Obj2Map(Object obj) {
		Map<String, String> hashMap = new HashMap<String, String>();
		try {
			Class<? extends Object> c = obj.getClass();
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

	static class CommonComparator implements Comparator<Map<String, String>>
	{
		private String key;
	    private String keyType;

	    public CommonComparator(String key, String keyType)
	    {
	    	this.key = key;
	        this.keyType = keyType;
	    }

	    public int compare(Map<String, String> first,
	                       Map<String, String> second)
	    {
	        // TODO: Null checking, both for maps and values
	    	if ((first == null) && (second == null))
	    		return 0;
	    	if (first == null)
	    		throw new RuntimeException("first object is null in CommonComparator");
	    	if (second == null)
	    		throw new RuntimeException("second object is null in CommonComparator");
	    	
	        String firstValue = first.get(key);
	        String secondValue = second.get(key);
	        if ((firstValue == null) && (secondValue == null))
	    		return 0;
	        if (firstValue == null)
	    		throw new RuntimeException("firstValue is null in CommonComparator");
	    	if (secondValue == null)
	    		throw new RuntimeException("secondValue is null in CommonComparator");
	        
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
	static class DateTimeComparator implements Comparator<Map<String, String>>
	{
		private String key1;
		private String key2;
	    private String keyType;

	    public DateTimeComparator(String key1, String key2, String keyType)
	    {
	    	this.key1 = key1;
	    	this.key2 = key2;
	        this.keyType = keyType;
	    }

	    public int compare(Map<String, String> first,
	                       Map<String, String> second)
	    {
	        // TODO: Null checking, both for maps and values
	    	if ((first == null) && (second == null))
	    		return 0;
	    	if (first == null)
	    		throw new RuntimeException("first object is null in DateTimeComparator");
	    	if (second == null)
	    		throw new RuntimeException("second object is null in DateTimeComparator");
	    	
	        String firstValue_key1 = first.get(key1);
	        String firstValue_key2 = first.get(key2);
	        String secondValue_key1 = second.get(key1);
	        String secondValue_key2 = second.get(key2);
	        
	        if ((firstValue_key1 == null) || (firstValue_key2 == null))
	    		throw new RuntimeException("firstValue is null in DateTimeComparator");
	        if ((secondValue_key1 == null) || (secondValue_key2 == null))
	    		throw new RuntimeException("secondValue is null in CommonComparator");
	        
	        if(keyType.equals("DateTime")){ //assume date in key1 and time in key2

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
		    valA = a.get(fieldName);
		    valB = b.get(fieldName);

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
			 Class<?> class_obj=null;
			 if (class_type.equals("trigger_Memory")) //only one metricType supported now
				 class_obj = PolicyTrigger.class;
			 for (String key : range.keySet()) {
				 Map<String, Map<String, String>> value = range.get(key);
				 for(String value_key : value.keySet()){
					 Map<String, String> value_item = value.get(value_key);
					 ConstraintMapping mapping = config.createConstraintMapping();
					 if (value_key.endsWith("Min")){
						 mapping.type(class_obj).property(key, ElementType.FIELD).constraint( new MinDef().value(Integer.parseInt(value_item.get("value"))).message(value_item.get("message")));
					 }
					 else if(value_key.endsWith("Max")){
						 mapping.type(class_obj).property(key, ElementType.FIELD).constraint( new MaxDef().value(Integer.parseInt(value_item.get("value"))).message(value_item.get("message")));
					 }
					 else if(value_key.endsWith("NotNull")){
						 mapping.type(class_obj).property(key, ElementType.FIELD).constraint( new NotNullDef().message(value_item.get("message")));
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
					Iterator<Entry<String, JsonNode>> trigger_items= subNode.fields();
					while (trigger_items.hasNext()){
						Entry<String, JsonNode> item_node = trigger_items.next();
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
	    	last_timestamp = last_history.get("startTime").asLong();
	    }
	    else {
	    	sub_list = elements_sorted;
	    	last_timestamp = 0;
	    }

	    JsonNodeFactory factory = JsonNodeFactory.instance;
	    ArrayNode aaData = new ArrayNode(factory);
	    aaData.addAll(sub_list);
		jNode.set("data", aaData);
	    jNode.put("timestamp", last_timestamp);
		return jNode.toString();

	}



	public static JsonNode parsePolicy(String jsonString, Map<String, String> service_info, HttpServletRequest httpServletRequest) throws JsonParseException, JsonMappingException, IOException{
		 List<String> violation_message = new ArrayList<String>();
		 ObjectNode result = new_mapper.createObjectNode();
		 result.put("valid", false);
		 logger.info("received policy : " + jsonString); //debug
		 new_mapper.readValue(jsonString, Policy.class); //for syntax error check
		 String transfered_json = TransferedPolicy.packServiceInfo(jsonString, service_info);
		 logger.info("transfered policy after update with service_information : " + transfered_json); //debug
		 TransferedPolicy policy = new_mapper.readValue(transfered_json, TransferedPolicy.class);
		 logger.info("we get policy as " + (Obj2Map(policy)).toString());//debug
		 //additional data manipulation and check again
		 policy = policy.setMaxInstCount();
		 HibernateValidatorConfiguration config = BeanValidation.getPolicyRange();
		 ValidatorFactory vf = config.buildValidatorFactory();
		 Locale locale = LocaleUtil.getLocale(httpServletRequest);
		 MessageInterpolator interpolator = new LocaleSpecificMessageInterpolator(vf.getMessageInterpolator(), locale);
		 Validator validator = vf.usingContext().messageInterpolator(interpolator).getValidator();
		 Set<ConstraintViolation<TransferedPolicy>> set = validator.validate(policy);
		 if (set.size() > 0 ){
			 for (ConstraintViolation<TransferedPolicy> constraintViolation : set) {
				 violation_message.add(constraintViolation.getMessage());
			 }
			 result.set("violation_message", new_mapper.valueToTree(violation_message));
			 return result;
		 }

		 String new_json = policy.transformInput();
		 logger.info("policy before trigger back : " + new_json); //debug
		 new_json = TransferedPolicy.unpackServiceInfo(new_json, service_info);
		 result.put("valid", true);
		 logger.info("send out policy: " + new_json); //debug
		 result.put("new_json", new_json);
		 return result;
	}

	public static JsonNode parsePolicyOutput(String jsonString, Map<String, String> supplyment, Map<String, String> service_info, HttpServletRequest httpServletRequest) throws JsonParseException, JsonMappingException, IOException{
		 List<String> violation_message = new ArrayList<String>();
		 ObjectNode result = new_mapper.createObjectNode();
		 result.put("valid", false);
		 logger.info("received json: " + jsonString); 
		 String transfered_json = TransferedPolicy.packServiceInfo(jsonString, service_info);
		 logger.info("transfered policy after update with service_information : " + transfered_json); //debug
		 TransferedPolicy policy = new_mapper.readValue(transfered_json, TransferedPolicy.class);
		 logger.info("we get policy as " + (Obj2Map(policy)).toString());//debug
		 ValidatorFactory vf = Validation.buildDefaultValidatorFactory();
		 Locale locale = LocaleUtil.getLocale(httpServletRequest);
		 MessageInterpolator interpolator = new LocaleSpecificMessageInterpolator(vf.getMessageInterpolator(), locale);
		 Validator validator = vf.usingContext().messageInterpolator(interpolator).getValidator();
		 Set<ConstraintViolation<TransferedPolicy>> set = validator.validate(policy);
		 if (set.size() > 0 ){
			 for (ConstraintViolation<TransferedPolicy> constraintViolation : set) {
				 violation_message.add(constraintViolation.getMessage());
			 }
			 result.set("violation_message", new_mapper.valueToTree(violation_message));
			 return result;
		 }

		 //additional data manipulation
		 policy = policy.transformSchedules();
		 String new_json = policy.transformOutput(supplyment, service_info);
		 result.put("valid", true);
		 result.put("new_json", new_json);
		 return result;
	}

	public static JsonNode parsePolicyEnable(String jsonString, HttpServletRequest httpServletRequest) throws JsonParseException, JsonMappingException, IOException{
		 List<String> violation_message = new ArrayList<String>();
		 ObjectNode result = new_mapper.createObjectNode();
		 result.put("valid", false);
		 PolicyEnbale policyEnable = new_mapper.readValue(jsonString, PolicyEnbale.class);
		 ValidatorFactory vf = Validation.buildDefaultValidatorFactory();
		 Locale locale = LocaleUtil.getLocale(httpServletRequest);
		 MessageInterpolator interpolator = new LocaleSpecificMessageInterpolator(vf.getMessageInterpolator(), locale);
		 Validator validator = vf.usingContext().messageInterpolator(interpolator).getValidator();
		 Set<ConstraintViolation<PolicyEnbale>> set = validator.validate(policyEnable);
		 if (set.size() > 0 ){
			 for (ConstraintViolation<PolicyEnbale> constraintViolation : set) {
				 violation_message.add(constraintViolation.getMessage());
			 }
			 result.set("violation_message", new_mapper.valueToTree(violation_message));
			 return result;
		 }

		 //additional data manipulation
		 String new_json = policyEnable.transformInput();
		 result.put("valid", true);
		 result.put("new_json", new_json);
		 return result;
	}

	public static JavaType  getCollectionType(Class<?> parametrized, Class<?> parametersFor, Class<?>... parameterTypes) {
        return new_mapper.getTypeFactory().constructParametrizedType(parametrized, parametersFor, parameterTypes);
	}

	public static JsonNode parseScalingHistory(String jsonString, HttpServletRequest httpServletRequest) throws JsonParseException, JsonMappingException, IOException{
		 List<String> violation_message = new ArrayList<String>();
		 ObjectNode result = new_mapper.createObjectNode();
		 result.put("valid", false);
		 JavaType javaType = getCollectionType(ArrayList.class, ArrayList.class, HistoryData.class);
		 new_mapper.configure(DeserializationFeature.ACCEPT_SINGLE_VALUE_AS_ARRAY, true);
		 List<HistoryData> scalinghistory = (List<HistoryData>)new_mapper.readValue(jsonString, javaType);
		 ValidatorFactory vf = Validation.buildDefaultValidatorFactory();
		 Locale locale = LocaleUtil.getLocale(httpServletRequest);
		 MessageInterpolator interpolator = new LocaleSpecificMessageInterpolator(vf.getMessageInterpolator(), locale);
		 Validator validator = vf.usingContext().messageInterpolator(interpolator).getValidator();
		 Set<ConstraintViolation<List<HistoryData>>> set = validator.validate(scalinghistory);
		 if (set.size() > 0 ){
			 for (ConstraintViolation<List<HistoryData>> constraintViolation : set) {
				 violation_message.add(constraintViolation.getMessage());
			 }
			 result.set("violation_message", new_mapper.valueToTree(violation_message));
			 return result;
		 }

		 //additional data manipulation
     	 String new_json = transformHistory(scalinghistory);
		 result.put("valid", true);
		 result.put("new_json", new_json);
		 return result;
	}

	public static JsonNode parseMetrics(String jsonString, HttpServletRequest httpServletRequest) throws JsonParseException, JsonMappingException, IOException{
		 List<String> violation_message = new ArrayList<String>();
		 ObjectNode result = new_mapper.createObjectNode();
		 result.put("valid", false);
		 //JavaType javaType = getCollectionType(ArrayList.class, HistoryData.class);
		 //new_mapper.configure(DeserializationFeature.ACCEPT_SINGLE_VALUE_AS_ARRAY, true);
		 logger.info("Received metrics: " + jsonString);
		 Metrics metrics = new_mapper.readValue(jsonString, Metrics.class);
		 
		 ValidatorFactory vf = Validation.buildDefaultValidatorFactory();
		 Locale locale = LocaleUtil.getLocale(httpServletRequest);
		 MessageInterpolator interpolator = new LocaleSpecificMessageInterpolator(vf.getMessageInterpolator(), locale);
		 Validator validator = vf.usingContext().messageInterpolator(interpolator).getValidator();
		 Set<ConstraintViolation<Metrics>> set = validator.validate(metrics);
		 if (set.size() > 0 ){
			 for (ConstraintViolation<Metrics> constraintViolation : set) {
				 violation_message.add(constraintViolation.getMessage());
			 }
			 result.set("violation_message", new_mapper.valueToTree(violation_message));
			 return result;
		 }
          

		 //additional data manipulation
    	 String new_json = metrics.transformOutput();
		 result.put("valid", true);
		 result.put("new_json", new_json);
		 return result;
	}

}
