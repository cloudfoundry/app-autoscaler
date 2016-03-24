package org.cloudfoundry.autoscaler.util;

import java.io.IOException;

import org.cloudfoundry.autoscaler.data.couchdb.document.AutoScalerPolicy;

import com.fasterxml.jackson.core.JsonParseException;
import com.fasterxml.jackson.databind.JsonMappingException;
import com.fasterxml.jackson.databind.ObjectMapper;

/**
 * This class is used to parse policy JSON string.
 * 
 *
 */
public class PolicyParser {
	private static final ObjectMapper mapper = new ObjectMapper();
	
	public static AutoScalerPolicy parse(String jsonString) throws JsonParseException, JsonMappingException, IOException{
		return mapper.readValue(jsonString, AutoScalerPolicy.class);
	}
}
