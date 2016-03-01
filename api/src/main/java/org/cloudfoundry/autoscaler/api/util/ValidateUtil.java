package org.cloudfoundry.autoscaler.api.util;

import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.Comparator;
import java.util.Date;
import java.util.HashMap;
import java.util.HashSet;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.Set;

import javax.servlet.http.HttpServletRequest;

import org.apache.log4j.Logger;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.node.ObjectNode;
import com.fasterxml.jackson.databind.exc.InvalidFormatException;
import com.fasterxml.jackson.databind.JsonMappingException;
import com.fasterxml.jackson.core.JsonParseException;
import com.fasterxml.jackson.core.JsonProcessingException;

import org.cloudfoundry.autoscaler.api.util.ProtoType;
import org.cloudfoundry.autoscaler.api.exceptions.InputJSONParseErrorException;
import org.cloudfoundry.autoscaler.api.exceptions.InputJSONFormatErrorException;
import org.cloudfoundry.autoscaler.api.exceptions.OutputJSONFormatErrorException;
import org.cloudfoundry.autoscaler.api.exceptions.OutputJSONParseErrorException;

public class ValidateUtil {
	private static final String CLASS_NAME = ValidateUtil.class.getName();
	private static final Logger logger     = Logger.getLogger(CLASS_NAME);
	private static ObjectMapper objectMapper = new ObjectMapper();
	public enum DataType {
		CREATE_REQUEST, CREATE_RESPONSE,DELETE_REQUEST, DELETE_RESPONSE,GET_REQUEST, GET_RESPONSE,ENABLE_REQUEST, 
		ENABLE_RESPONSE, GET_STATUS_REQUEST, GET_STATUS_RESPONSE, GET_HISTORY_REQUEST, GET_HISTORY_RESPONSE, 
		GET_METRICS_REQUEST, GET_METRICS_RESPONSE
	};

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
	        	try {
	        	    if(dateformat.parse(firstValue).before(dateformat.parse(secondValue))){
	        		    return -1;
	        	    } else if (dateformat.parse(firstValue).equals(dateformat.parse(secondValue))){
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
	        	/*
	        	SimpleDateFormat dateformat = new SimpleDateFormat("yyyy-MM-dd");
	        	try {
	        	    if(dateformat.parse(firstValue_key1).before(dateformat.parse(secondValue_key1))){
	        	    	datecompare = -1;
	        	    } else if (dateformat.parse(firstValue_key1).equals(dateformat.parse(secondValue_key1))){
	        	    	datecompare = 0;
	        	    } else {
	        	    	datecompare = 1;
	        	    }
	        	} catch (Exception e){
	        		return -2;
	        	}
	        	
	        	int firsthour = Integer.parseInt((firstValue_key2.split(":"))[0]);  
	        	int firstminute = Integer.parseInt((firstValue_key2.split(":"))[1]);
	        	int secondhour = Integer.parseInt((secondValue_key2.split(":"))[0]);  
	        	int secondminute = Integer.parseInt((secondValue_key2.split(":"))[1]);
	        	
	        	if((firsthour == secondhour) && (firstminute == secondminute)){
	        		timecompare = 0;
	        	}
	        	else if((firsthour < secondhour) || ( (firsthour == secondhour ) && (firstminute < secondminute) )){
	        		timecompare = -1;
	        	}
	        	else {
	        		timecompare = 1;
	        	}
	        	
	        	if (datecompare != 0)
	        		return datecompare;
	        	else
	        		return timecompare;
	        	*/
 	        	SimpleDateFormat dateformat = new SimpleDateFormat("yyyy-MM-dd HH:mm");
 	        	String firstDateTime = firstValue_key1 + " " + firstValue_key2;
 	        	String secondDateTime = secondValue_key1 + " " + secondValue_key2;
 	        	
 	        	try {
 	        	if(dateformat.parse(firstDateTime).before(dateformat.parse(secondDateTime)))
 	        		return -1;
 	        	else if (dateformat.parse(firstDateTime).equals(dateformat.parse(secondDateTime)))
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
	
	public static boolean isNull(String str){
		if (str == null || str.trim().length() == 0)
			return true;
		return false;
	}
    
    //Java Bean Validation based validation and transformation
    public static Map<String, String> handleInput(DataType datatype, String jsonData, Map<String, String> service_info, HttpServletRequest httpServletRequest) throws Exception {
		Map<String, String> result = new HashMap<String, String>();
		result.put("result", "FAIL");
		String new_json="";
		//JsonNode newNode = objectMapper.readTree("");
		
		
		if(datatype == DataType.CREATE_REQUEST){
		    try {
				Map<String, Object> parse_result = ProtoType.parsePolicy(jsonData, service_info, httpServletRequest);
				if (Boolean.valueOf(parse_result.get("valid").toString()) == false) {
					List<String> violation_message = (List<String>)parse_result.get("violation_message");
					if (violation_message.size() > 0 ) {
						//result.put("error_message", "Violation failed with message: " + violation_message.get(0));
						//return result;
						throw new InputJSONFormatErrorException(MessageUtil.RestResponseErrorMsg_Create_Update_Policy_context, violation_message.get(0));
					}
				}
				new_json = (String)parse_result.get("new_json");
		    } 
		    catch (InputJSONFormatErrorException e) {
		    	throw e;
		    }
		    catch (JsonMappingException | JsonParseException e) {
		    	logger.info("Jackson throw out the expection type of " + e.getClass().getName());
				logger.info("Validate policy json data raise exception: " + e.getMessage());
				JsonProcessingException exc = (JsonProcessingException)e;
				logger.info("!!!!!!Input json error at Line : " + exc.getLocation().getLineNr() + " and Column: " + exc.getLocation().getColumnNr());
				throw new InputJSONFormatErrorException(MessageUtil.RestResponseErrorMsg_Create_Update_Policy_context, e.getMessage(), exc.getLocation().getLineNr(), exc.getLocation().getColumnNr());
		    	
		    }
		    catch (Exception e) {
		    	logger.info("Validate policy json data throw out the expection type of " + e.getClass().getName());
				logger.info("Validate policy json data raise exception: " + e.getMessage());
			    throw new InputJSONParseErrorException(MessageUtil.RestResponseErrorMsg_Create_Update_Policy_context, e.getMessage());
			}
		}
		else if (datatype == DataType.ENABLE_REQUEST){
			try {
				Map<String, Object> parse_result = ProtoType.parsePolicyEnable(jsonData, httpServletRequest);
				if (Boolean.valueOf(parse_result.get("valid").toString()) == false) {
					List<String> violation_message = (List<String>)parse_result.get("violation_message");
					if (violation_message.size() > 0 ) {
						//result.put("error_message", "Violation failed with message: " + violation_message.get(0));
						//return result;
						throw new InputJSONFormatErrorException(MessageUtil.RestResponseErrorMsg_Enable_Policy_context, violation_message.get(0));
					}
				}
				new_json = (String)parse_result.get("new_json");
			}
		    catch (InputJSONFormatErrorException e) {
		    	throw e;
		    }
		    catch (JsonMappingException | JsonParseException e) {
		    	logger.info("Jackson throw out the expection type of " + e.getClass().getName());
				logger.info("Validate policy json data raise exception: " + e.getMessage());
				JsonProcessingException exc = (JsonProcessingException)e;
				logger.info("!!!!!!Input json error at Line : " + exc.getLocation().getLineNr() + " and Column: " + exc.getLocation().getColumnNr());
				throw new InputJSONFormatErrorException(MessageUtil.RestResponseErrorMsg_Enable_Policy_context, e.getMessage(), exc.getLocation().getLineNr(), exc.getLocation().getColumnNr());
		    	
		    }
			catch (Exception e) {
		    	logger.info("Validate policy json data throw out the expection type of " + e.getClass().getName());
				logger.info("Validate policy json data raise exception: " + e.getMessage());
				throw new InputJSONParseErrorException(MessageUtil.RestResponseErrorMsg_Enable_Policy_context, e.getMessage());
			}			
		}
		
		result.put("result", "OK");
		result.put("json", new_json);
		return result;
    }
    
    public static Map<String, String> handleOutput(DataType datatype, String jsonData, Map<String, String> supplyment, Map<String, String> service_info, HttpServletRequest httpServletRequest) throws Exception {
		Map<String, String> result = new HashMap<String, String>();
		result.put("result", "FAIL");
		String new_json="";
		
		//should define where to bypass the validation, like in this case
		if (datatype == DataType.GET_RESPONSE){
			try {
				Map<String, Object> parse_result = ProtoType.parsePolicyOutput(jsonData, supplyment, service_info, httpServletRequest);
				if (Boolean.valueOf(parse_result.get("valid").toString()) == false) {
					List<String> violation_message = (List<String>)parse_result.get("violation_message");
					if (violation_message.size() > 0 ) {
						//result.put("error_message", "Violation failed with message: " + violation_message.get(0));
						//return result;
						throw new OutputJSONFormatErrorException(MessageUtil.RestResponseErrorMsg_Get_Policy_context, violation_message.get(0));
					}
				}
				new_json = (String)parse_result.get("new_json");
			}  
		    catch (OutputJSONFormatErrorException e) {
		    	throw e;
		    }
			catch (Exception e) {
				logger.info("Jackson throw out the expection type of " + e.getClass().getName());
				logger.info("Validate policy json data raise exception: " + e.getMessage());
				if (e instanceof InvalidFormatException) {
					throw new OutputJSONFormatErrorException(MessageUtil.RestResponseErrorMsg_Get_Policy_context, e.getMessage());
				}
				else {
				//result.put("error_message", "can not parse json data for get policy ");
				//return result;
				    throw new OutputJSONParseErrorException(MessageUtil.RestResponseErrorMsg_Get_Policy_context, e.getMessage());
				}
			}			 
		} else if (datatype == DataType.GET_HISTORY_RESPONSE){
			try {
				Map<String, Object> parse_result = ProtoType.parseScalingHistory(jsonData, httpServletRequest);
				if (Boolean.valueOf(parse_result.get("valid").toString()) == false) {
					List<String> violation_message = (List<String>)parse_result.get("violation_message");
					if (violation_message.size() > 0 ) {
						//result.put("error_message", "Violation failed with message: " + violation_message.get(0));
						//return result;
						throw new OutputJSONFormatErrorException(MessageUtil.RestResponseErrorMsg_Get_Scaling_History_context, violation_message.get(0));
					}
				}
				new_json = (String)parse_result.get("new_json");
			}  
		    catch (OutputJSONFormatErrorException e) {
		    	throw e;
		    } 
			catch (Exception e) {
				logger.info("Jackson throw out the expection type of " + e.getClass().getName());
				logger.info("Validate history json data raise exception: " + e.getMessage());
				if (e instanceof InvalidFormatException) {
					throw new OutputJSONFormatErrorException(MessageUtil.RestResponseErrorMsg_Get_Scaling_History_context, e.getMessage());
				}
				else {
				//result.put("error_message", "can not parse json data for get Scaling History ");
				//return result;
				    throw new OutputJSONParseErrorException(MessageUtil.RestResponseErrorMsg_Get_Scaling_History_context, e.getMessage());
				}
			}			 
		} else if (datatype == DataType.GET_METRICS_RESPONSE){
			try {
				Map<String, Object> parse_result = ProtoType.parseMetrics(jsonData, httpServletRequest);
				if (Boolean.valueOf(parse_result.get("valid").toString()) == false) {
					List<String> violation_message = (List<String>)parse_result.get("violation_message");
					if (violation_message.size() > 0 ) {
						//result.put("error_message", "Violation failed with message: " + violation_message.get(0));
						//return result;
						throw new OutputJSONFormatErrorException(MessageUtil.RestResponseErrorMsg_Get_Metric_Data_context, violation_message.get(0));
					}
				}
				new_json = (String)parse_result.get("new_json");
			}   
		    catch (OutputJSONFormatErrorException e) {
		    	throw e;
		    }
			catch (Exception e) {
				logger.info("Jackson throw out the expection type of " + e.getClass().getName());
				logger.info("Validate metric json data raise exception: " + e.getMessage());
				if (e instanceof InvalidFormatException) {
					throw new OutputJSONFormatErrorException(MessageUtil.RestResponseErrorMsg_Get_Metric_Data_context, e.getMessage());
				}
				else {
				//result.put("error_message", "can not parse json data for get Metric data ");
				//return result;
				    throw new OutputJSONParseErrorException(MessageUtil.RestResponseErrorMsg_Get_Metric_Data_context, e.getMessage());
				}
			}			 
		}
		

		result.put("result", "OK");
		result.put("json", new_json);
		return result;
    }
    
}
