package org.cloudfoundry.autoscaler.data;

import java.io.IOException;
import java.io.InputStream;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.AutoScalerPolicy;
import org.cloudfoundry.autoscaler.exceptions.PolicyNotFoundException;

import com.fasterxml.jackson.core.JsonParseException;
import com.fasterxml.jackson.databind.JsonMappingException;
import com.fasterxml.jackson.databind.ObjectMapper;

public class DefaultAutoScalerPolicy {
	private static AutoScalerPolicy defaultPolicy  = null;
	private static final ObjectMapper mapper = new ObjectMapper();
	public static final String DEFAULT_POLICY = "default";
	/**
	 * Gets default policy
	 * @return
	 * @throws JsonParseException
	 * @throws JsonMappingException
	 * @throws IOException
	 */
	public static AutoScalerPolicy getDefaultPolicy() throws PolicyNotFoundException{
		if (defaultPolicy != null)
			return defaultPolicy;
		ClassLoader loader = DefaultAutoScalerPolicy.class.getClassLoader();
		InputStream stream = loader.getResourceAsStream("defaultPolicy.json");
		try {
			defaultPolicy = mapper.readValue(stream, AutoScalerPolicy.class);
			defaultPolicy.setPolicyId(DEFAULT_POLICY);
		} catch (JsonParseException e) {
			throw new PolicyNotFoundException(DEFAULT_POLICY, e.getCause());
		} catch (JsonMappingException e) {
			throw new PolicyNotFoundException(DEFAULT_POLICY, e.getCause());
		} catch (IOException e) {
			throw new PolicyNotFoundException(DEFAULT_POLICY, e.getCause());
		}
		return defaultPolicy;
	}
	
}
