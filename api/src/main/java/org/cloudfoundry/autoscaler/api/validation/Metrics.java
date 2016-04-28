package org.cloudfoundry.autoscaler.api.validation;

import java.io.IOException;
import java.util.ArrayList;
import java.util.Collections;
import java.util.Iterator;
import java.util.List;

import javax.validation.Valid;

import org.cloudfoundry.autoscaler.api.validation.BeanValidation.JsonObjectComparator;
import org.cloudfoundry.autoscaler.common.util.ConfigManager;

import com.fasterxml.jackson.core.JsonParseException;
import com.fasterxml.jackson.databind.JsonMappingException;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.node.ArrayNode;
import com.fasterxml.jackson.databind.node.JsonNodeFactory;
import com.fasterxml.jackson.databind.node.ObjectNode;

public class Metrics {
	@Valid
	private List<MetricData> data;

	public List<MetricData> getData() {
		return this.data;
	}

	public void setData(List<MetricData> data) {
		this.data = data;
	}

	public String transformOutput() throws JsonParseException, JsonMappingException, IOException{
		String current_json =  BeanValidation.new_mapper.writeValueAsString(this.data);
		JsonNode top = BeanValidation.new_mapper.readTree(current_json);
		JsonObjectComparator comparator = new JsonObjectComparator("timestamp", "long");
		Iterator<JsonNode> elements = top.elements();
		List<JsonNode> elements_sorted = new ArrayList<JsonNode>();
		while(elements.hasNext()){
			elements_sorted.add(elements.next());
		}
		Collections.sort(elements_sorted, comparator);

		ObjectNode jNode = BeanValidation.new_mapper.createObjectNode();

		List<JsonNode> sub_list;
	    long last_timestamp;
	    int max_len = Integer.parseInt(ConfigManager.get("maxMetricRecord"));
	    BeanValidation.logger.info("Current maxMetricRecord returned is " + max_len + " and current number of metric record is " + elements_sorted.size());
	    if(elements_sorted.size() > max_len){
	    	sub_list = elements_sorted.subList(0, max_len);
	    	JsonNode last_metric = elements_sorted.get(max_len-1);
	    	last_timestamp = last_metric.get("timestamp").asLong();
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
}