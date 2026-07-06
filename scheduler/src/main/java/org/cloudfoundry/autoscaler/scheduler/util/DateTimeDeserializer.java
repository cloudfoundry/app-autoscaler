package org.cloudfoundry.autoscaler.scheduler.util;

import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;
import tools.jackson.core.JsonParser;
import tools.jackson.databind.DeserializationContext;
import tools.jackson.databind.ValueDeserializer;

public class DateTimeDeserializer extends ValueDeserializer<LocalDateTime> {

  @Override
  public LocalDateTime deserialize(JsonParser parser, DeserializationContext ctxt) {
    DateTimeFormatter dateTimeFormatter = DateTimeFormatter.ofPattern(DateHelper.DATE_TIME_FORMAT);
    return LocalDateTime.parse(parser.getValueAsString(), dateTimeFormatter);
  }
}
