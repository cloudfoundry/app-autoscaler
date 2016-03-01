package org.cloudfoundry.autoscaler.servicebroker.util;

import org.cloudfoundry.autoscaler.servicebroker.Constants;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;

public class LoggingUtil {

    private static final ObjectMapper mapper= Constants.MAPPER;

	public static final String SEPREATOR = ",";
	public static final String ARROW = "===>";
	public static final String CONFLICIT = "<===>";
	public static final String LBRACKET = "[";
	public static final String RBRACKET = "]";

	    
    public static String writeObjectArrayAsString (Object[] objs){
		if (objs != null) {
			StringBuilder str = new StringBuilder();
			for (Object obj : objs) {
				try {
					str.append(mapper.writeValueAsString(obj)).append(",");
				} catch (JsonProcessingException e) {
					str.append(obj.toString()).append(",");
				}
			}
			return addBracketWrapper(str.toString());
		} 
		return null;
    }    
    
    public static String writeObjectAsString (Object obj){
		if (obj != null) {
			try {
				return addBracketWrapper(mapper.writeValueAsString(obj));
			} catch (JsonProcessingException e) {
				return addBracketWrapper(obj.toString());
			}
		} 
		return null;
    }    
    
    public static String addBracketWrapper (Object obj) {
    	
    	return (obj ==null) ? 
    			new StringBuilder().append(LBRACKET).append("null").append(RBRACKET).toString(): 
    			new StringBuilder().append(LBRACKET).append(obj).append(RBRACKET).toString();
    }  

    

	
}
