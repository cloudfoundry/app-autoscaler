package org.cloudfoundry.autoscaler.api.validation;

import java.util.HashMap;
import java.util.Map;

import javax.servlet.http.HttpServletRequest;

import org.apache.log4j.Logger;

import com.fasterxml.jackson.databind.exc.InvalidFormatException;
import com.fasterxml.jackson.databind.JsonMappingException;
import com.fasterxml.jackson.core.JsonParseException;
import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.JsonNode;

import org.cloudfoundry.autoscaler.api.util.MessageUtil;
import org.cloudfoundry.autoscaler.common.InputJSONFormatErrorException;
import org.cloudfoundry.autoscaler.common.InputJSONParseErrorException;
import org.cloudfoundry.autoscaler.common.OutputJSONFormatErrorException;
import org.cloudfoundry.autoscaler.common.OutputJSONParseErrorException;

public class ValidateUtil {
	private static final String CLASS_NAME = ValidateUtil.class.getName();
	private static final Logger logger     = Logger.getLogger(CLASS_NAME);
	public enum DataType {
		CREATE_REQUEST, CREATE_RESPONSE,DELETE_REQUEST, DELETE_RESPONSE,GET_REQUEST, GET_RESPONSE,ENABLE_REQUEST, 
		ENABLE_RESPONSE, GET_STATUS_REQUEST, GET_STATUS_RESPONSE, GET_HISTORY_REQUEST, GET_HISTORY_RESPONSE, 
		GET_METRICS_REQUEST, GET_METRICS_RESPONSE
	};

	
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
		
		
		if(datatype == DataType.CREATE_REQUEST){
		    try {
				JsonNode parse_result = BeanValidation.parsePolicy(jsonData, service_info, httpServletRequest);
				if (parse_result.get("valid").asBoolean() == false) {
					JsonNode violation_message = parse_result.get("violation_message");
					if ((violation_message.isArray()) && (violation_message.size() > 0 )) {
						throw new InputJSONFormatErrorException(MessageUtil.RestResponseErrorMsg_Create_Update_Policy_context, violation_message.get(0).asText());
					}
				}
				new_json = parse_result.get("new_json").asText();
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
				JsonNode parse_result = BeanValidation.parsePolicyEnable(jsonData, httpServletRequest);
				if (parse_result.get("valid").asBoolean() == false) {
					JsonNode violation_message = parse_result.get("violation_message");
					if ((violation_message.isArray()) && (violation_message.size() > 0 )) {
						throw new InputJSONFormatErrorException(MessageUtil.RestResponseErrorMsg_Enable_Policy_context, violation_message.get(0).asText());
					}
				}
				new_json = parse_result.get("new_json").asText();
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
				JsonNode parse_result = BeanValidation.parsePolicyOutput(jsonData, supplyment, service_info, httpServletRequest);
				if (parse_result.get("valid").asBoolean() == false) {
					JsonNode violation_message = parse_result.get("violation_message");
					if ((violation_message.isArray()) && (violation_message.size() > 0 )) {
						throw new OutputJSONFormatErrorException(MessageUtil.RestResponseErrorMsg_Get_Policy_context, violation_message.get(0).asText());
					}
				}
				new_json = parse_result.get("new_json").asText();
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
				    throw new OutputJSONParseErrorException(MessageUtil.RestResponseErrorMsg_Get_Policy_context, e.getMessage());
				}
			}			 
		} else if (datatype == DataType.GET_HISTORY_RESPONSE){
			try {
				JsonNode parse_result = BeanValidation.parseScalingHistory(jsonData, httpServletRequest);
				if (parse_result.get("valid").asBoolean() == false) {
					JsonNode violation_message = parse_result.get("violation_message");
					if ((violation_message.isArray()) && (violation_message.size() > 0 )) {
						throw new OutputJSONFormatErrorException(MessageUtil.RestResponseErrorMsg_Get_Scaling_History_context, violation_message.get(0).asText());
					}
				}
				new_json = parse_result.get("new_json").asText();
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
				    throw new OutputJSONParseErrorException(MessageUtil.RestResponseErrorMsg_Get_Scaling_History_context, e.getMessage());
				}
			}			 
		} else if (datatype == DataType.GET_METRICS_RESPONSE){
			try {
				JsonNode parse_result = BeanValidation.parseMetrics(jsonData, httpServletRequest);
				if (Boolean.valueOf(parse_result.get("valid").toString()) == false) {
					JsonNode violation_message = parse_result.get("violation_message");
					if ((violation_message.isArray()) && (violation_message.size() > 0 )) {
						throw new OutputJSONFormatErrorException(MessageUtil.RestResponseErrorMsg_Get_Metric_Data_context, violation_message.get(0).asText());
					}
				}
				new_json = parse_result.get("new_json").asText();
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
				    throw new OutputJSONParseErrorException(MessageUtil.RestResponseErrorMsg_Get_Metric_Data_context, e.getMessage());
				}
			}			 
		}
		

		result.put("result", "OK");
		result.put("json", new_json);
		return result;
    }
    
}
