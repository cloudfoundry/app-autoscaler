package org.cloudfoundry.autoscaler.scheduler.util;

import com.fasterxml.jackson.core.JsonParser;
import com.fasterxml.jackson.databind.DeserializationContext;
import com.fasterxml.jackson.databind.JsonDeserializer;
import java.io.IOException;
import java.time.LocalDate;
import java.time.format.DateTimeFormatter;

public class DateDeserializer extends JsonDeserializer<LocalDate> {

  @Override
  public LocalDate deserialize(JsonParser parser, DeserializationContext ctxt) throws IOException {
    DateTimeFormatter dateTimeFormatter = DateTimeFormatter.ofPattern(DateHelper.DATE_FORMAT);
    return LocalDate.parse(parser.getValueAsString(), dateTimeFormatter);
  }
}
