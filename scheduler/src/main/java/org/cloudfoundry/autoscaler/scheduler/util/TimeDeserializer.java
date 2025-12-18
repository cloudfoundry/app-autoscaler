package org.cloudfoundry.autoscaler.scheduler.util;

import tools.jackson.core.JsonParser;
import tools.jackson.databind.DeserializationContext;
import java.io.IOException;
import java.time.LocalTime;
import java.time.format.DateTimeFormatter;
import tools.jackson.databind.ValueDeserializer;

public class TimeDeserializer extends ValueDeserializer<LocalTime> {

  @Override
  public LocalTime deserialize(final JsonParser jp, final DeserializationContext ctxt) {
    DateTimeFormatter dateTimeFormatter = DateTimeFormatter.ofPattern(DateHelper.TIME_FORMAT);
    return LocalTime.parse(jp.getValueAsString(), dateTimeFormatter);
  }
}
