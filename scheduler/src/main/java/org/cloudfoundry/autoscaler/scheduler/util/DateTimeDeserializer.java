package org.cloudfoundry.autoscaler.scheduler.util;

import java.io.IOException;
import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.Date;

import com.fasterxml.jackson.core.JsonParser;
import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.DeserializationContext;
import com.fasterxml.jackson.databind.JsonDeserializer;

public class DateTimeDeserializer extends JsonDeserializer<Date> {

	@Override
	public Date deserialize(JsonParser parser, DeserializationContext ctxt)
			throws IOException, JsonProcessingException {
		SimpleDateFormat simpleDateFormat = new SimpleDateFormat(DateHelper.DATE_TIME_FORMAT);
		try {
			return simpleDateFormat.parse(parser.getValueAsString());
		} catch (ParseException e) {
			throw new IOException("Invalid DateTime can not parse: " + e.getMessage());
		}

	}

}
