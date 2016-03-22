package org.cloudfoundry.autoscaler.api.validation;

import java.io.IOException;
import java.util.HashMap;
import java.util.Map;

import javax.validation.constraints.NotNull;

import com.fasterxml.jackson.core.JsonParseException;
import com.fasterxml.jackson.databind.JsonMappingException;

public class PolicyEnbale {
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
		return  BeanValidation.new_mapper.writeValueAsString(result);
	}
}