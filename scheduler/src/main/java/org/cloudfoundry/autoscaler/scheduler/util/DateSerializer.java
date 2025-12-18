package org.cloudfoundry.autoscaler.scheduler.util;

import tools.jackson.core.JsonGenerator;
import java.io.IOException;
import java.time.LocalDate;
import java.time.format.DateTimeFormatter;
import tools.jackson.databind.SerializationContext;
import tools.jackson.databind.ValueSerializer;

public class DateSerializer extends ValueSerializer<LocalDate> {

  @Override
  public void serialize(LocalDate value, JsonGenerator gen, SerializationContext serializers) {
    DateTimeFormatter dateTimeFormatter = DateTimeFormatter.ofPattern(DateHelper.DATE_FORMAT);
    gen.writeString(value.format(dateTimeFormatter));
  }
}
