package org.cloudfoundry.autoscaler.scheduler.util;

import java.io.IOException;
import java.text.SimpleDateFormat;
import java.util.Date;

import com.fasterxml.jackson.core.JsonGenerator;
import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.JsonSerializer;
import com.fasterxml.jackson.databind.SerializerProvider;

public class DateSerializer extends JsonSerializer<Date> {

	@Override
	public void serialize(Date value, JsonGenerator gen, SerializerProvider serializers)
			throws IOException, JsonProcessingException {

		SimpleDateFormat dateFormat = new SimpleDateFormat(DateHelper.DATE_FORMAT);
		String formattedDate = dateFormat.format(value);
		gen.writeString(formattedDate);

	}


}
