package org.cloudfoundry.autoscaler.scheduler.util;

import tools.jackson.core.JsonParser;
import tools.jackson.databind.DeserializationContext;
import java.io.IOException;
import java.time.LocalDate;
import java.time.format.DateTimeFormatter;
import tools.jackson.databind.ValueDeserializer;

public class DateDeserializer extends ValueDeserializer<LocalDate> {

  @Override
  public LocalDate deserialize(JsonParser parser, DeserializationContext ctxt) {
    DateTimeFormatter dateTimeFormatter = DateTimeFormatter.ofPattern(DateHelper.DATE_FORMAT);
    return LocalDate.parse(parser.getValueAsString(), dateTimeFormatter);
  }
}
