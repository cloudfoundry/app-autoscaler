package org.cloudfoundry.autoscaler.scheduler.rest.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import io.swagger.v3.oas.annotations.media.Schema;
import jakarta.validation.constraints.Min;
import jakarta.validation.constraints.NotNull;

/** */
@Schema
public class ApplicationSchedules {
  @Schema(required = true)
  @NotNull
  @Min(value = 1)
  @JsonProperty(value = "instance_min_count")
  Integer instanceMinCount;

  @Schema(required = true)
  @NotNull
  @Min(value = 1)
  @JsonProperty(value = "instance_max_count")
  Integer instanceMaxCount;

  @Schema(required = true)
  @NotNull
  Schedules schedules;

  public Integer getInstanceMinCount() {
    return instanceMinCount;
  }

  public void setInstanceMinCount(Integer instanceMinCount) {
    this.instanceMinCount = instanceMinCount;
  }

  public Integer getInstanceMaxCount() {
    return instanceMaxCount;
  }

  public void setInstanceMaxCount(Integer instanceMaxCount) {
    this.instanceMaxCount = instanceMaxCount;
  }

  public Schedules getSchedules() {
    return schedules;
  }

  public void setSchedules(Schedules schedules) {
    this.schedules = schedules;
  }

  @Override
  public String toString() {
    return "ApplicationPolicy [instanceMinCount="
        + instanceMinCount
        + ", instanceMaxCount="
        + instanceMaxCount
        + ", schedules="
        + schedules
        + "]";
  }
}
