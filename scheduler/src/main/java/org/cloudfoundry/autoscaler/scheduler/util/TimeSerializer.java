package org.cloudfoundry.autoscaler.scheduler.util;

import com.fasterxml.jackson.core.JsonGenerator;
import com.fasterxml.jackson.databind.JsonSerializer;
import com.fasterxml.jackson.databind.SerializerProvider;
import java.io.IOException;
import java.time.LocalTime;
import java.time.format.DateTimeFormatter;

public class TimeSerializer extends JsonSerializer<LocalTime> {

  @Override
  public void serialize(LocalTime value, JsonGenerator gen, SerializerProvider serializers)
      throws IOException {
    DateTimeFormatter dateTimeFormatter = DateTimeFormatter.ofPattern(DateHelper.TIME_FORMAT);
    gen.writeString(value.format(dateTimeFormatter));
  }
}
