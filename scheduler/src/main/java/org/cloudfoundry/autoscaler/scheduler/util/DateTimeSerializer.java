package org.cloudfoundry.autoscaler.scheduler.util;

import tools.jackson.core.JsonGenerator;
import java.io.IOException;
import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;
import tools.jackson.databind.SerializationContext;
import tools.jackson.databind.ValueSerializer;

public class DateTimeSerializer extends ValueSerializer<LocalDateTime> {

  @Override
  public void serialize(LocalDateTime value, JsonGenerator gen, SerializationContext serializers) {
    DateTimeFormatter dateTimeFormatter = DateTimeFormatter.ofPattern(DateHelper.DATE_TIME_FORMAT);
    gen.writeString(value.format(dateTimeFormatter));
  }
}
