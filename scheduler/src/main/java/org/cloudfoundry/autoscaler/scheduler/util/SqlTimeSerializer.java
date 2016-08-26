package org.cloudfoundry.autoscaler.scheduler.util;

import java.io.IOException;
import java.sql.Time;
import java.text.SimpleDateFormat;

import com.fasterxml.jackson.core.JsonGenerator;
import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.JsonSerializer;
import com.fasterxml.jackson.databind.SerializerProvider;

public class SqlTimeSerializer extends JsonSerializer<Time> {

	@Override
	public void serialize(Time value, JsonGenerator gen, SerializerProvider serializers)
			throws IOException, JsonProcessingException {
		SimpleDateFormat dateFormat = new SimpleDateFormat(DateHelper.TIME_FORMAT);
		String formattedDate = dateFormat.format(value);
		gen.writeString(formattedDate);
	}

}
