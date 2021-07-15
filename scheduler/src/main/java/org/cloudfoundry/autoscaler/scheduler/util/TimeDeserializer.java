package org.cloudfoundry.autoscaler.scheduler.util;

import com.fasterxml.jackson.core.JsonParser;
import com.fasterxml.jackson.databind.DeserializationContext;
import com.fasterxml.jackson.databind.JsonDeserializer;
import java.io.IOException;
import java.time.LocalTime;
import java.time.format.DateTimeFormatter;

public class TimeDeserializer extends JsonDeserializer<LocalTime> {

  @Override
  public LocalTime deserialize(final JsonParser jp, final DeserializationContext ctxt)
      throws IOException {
    DateTimeFormatter dateTimeFormatter = DateTimeFormatter.ofPattern(DateHelper.TIME_FORMAT);
    return LocalTime.parse(jp.getValueAsString(), dateTimeFormatter);
  }
}
