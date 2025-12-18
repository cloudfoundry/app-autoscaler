package org.cloudfoundry.autoscaler.scheduler.util;

import tools.jackson.core.JsonGenerator;
import java.io.IOException;
import java.time.LocalTime;
import java.time.format.DateTimeFormatter;
import tools.jackson.databind.SerializationContext;
import tools.jackson.databind.ValueSerializer;

public class TimeSerializer extends ValueSerializer<LocalTime> {

  @Override
  public void serialize(LocalTime value, JsonGenerator gen, SerializationContext serializers) {
    DateTimeFormatter dateTimeFormatter = DateTimeFormatter.ofPattern(DateHelper.TIME_FORMAT);
    gen.writeString(value.format(dateTimeFormatter));
  }
}
