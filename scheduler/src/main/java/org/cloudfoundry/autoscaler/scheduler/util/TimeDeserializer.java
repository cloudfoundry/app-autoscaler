package org.cloudfoundry.autoscaler.scheduler.util;

import java.io.IOException;
import java.time.LocalTime;
import java.time.format.DateTimeFormatter;

import com.fasterxml.jackson.core.JsonParser;
import com.fasterxml.jackson.databind.DeserializationContext;
import com.fasterxml.jackson.databind.JsonDeserializer;

public class TimeDeserializer extends JsonDeserializer<LocalTime> {

	@Override
	public LocalTime deserialize(final JsonParser jp, final DeserializationContext ctxt) throws IOException {
		DateTimeFormatter dateTimeFormatter = DateTimeFormatter.ofPattern(DateHelper.TIME_FORMAT);
		return LocalTime.parse(jp.getValueAsString(), dateTimeFormatter);
	}
}
