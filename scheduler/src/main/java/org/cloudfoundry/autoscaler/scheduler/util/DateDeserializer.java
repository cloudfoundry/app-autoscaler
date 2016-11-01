package org.cloudfoundry.autoscaler.scheduler.util;

import java.io.IOException;
import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.Date;

import com.fasterxml.jackson.core.JsonParser;
import com.fasterxml.jackson.databind.DeserializationContext;
import com.fasterxml.jackson.databind.JsonDeserializer;

public class DateDeserializer extends JsonDeserializer<Date> {

	@Override
	public Date deserialize(JsonParser parser, DeserializationContext ctxt) throws IOException {
		SimpleDateFormat simpleDateFormat = new SimpleDateFormat(DateHelper.DATE_FORMAT);
		try {
			return simpleDateFormat.parse(parser.getValueAsString());
		} catch (ParseException e) {
			throw new IOException("Invalid Date can not parse: " + e.getMessage());
		}
	}

}
