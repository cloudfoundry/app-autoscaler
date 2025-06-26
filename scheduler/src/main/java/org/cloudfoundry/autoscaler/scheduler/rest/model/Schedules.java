package org.cloudfoundry.autoscaler.scheduler.rest.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import io.swagger.annotations.ApiModel;
import io.swagger.annotations.ApiModelProperty;
import jakarta.validation.constraints.NotBlank;
import java.util.List;
import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;

@ApiModel
public class Schedules {
  @ApiModelProperty(required = true, position = 1)
  @JsonProperty(value = "timezone")
  @NotBlank
  String timeZone;

  @ApiModelProperty(position = 3)
  @JsonProperty(value = "specific_date")
  private List<SpecificDateScheduleEntity> specificDate;

  @ApiModelProperty(position = 2)
  @JsonProperty(value = "recurring_schedule")
  private List<RecurringScheduleEntity> recurringSchedule;

  public boolean hasSchedules() {
    return this.hasRecurringSchedule() || this.hasSpecificDateSchedule();
  }

  public boolean hasRecurringSchedule() {
    if ((recurringSchedule == null || recurringSchedule.isEmpty())) {
      return false;
    }
    return true;
  }

  public boolean hasSpecificDateSchedule() {
    if ((specificDate == null || specificDate.isEmpty())) {
      return false;
    }
    return true;
  }

  public String getTimeZone() {
    return timeZone;
  }

  public void setTimeZone(String timeZone) {
    this.timeZone = timeZone;
  }

  public List<SpecificDateScheduleEntity> getSpecificDate() {
    return specificDate;
  }

  public void setSpecificDate(List<SpecificDateScheduleEntity> specificDate) {
    this.specificDate = specificDate;
  }

  public List<RecurringScheduleEntity> getRecurringSchedule() {
    return recurringSchedule;
  }

  public void setRecurringSchedule(List<RecurringScheduleEntity> recurringSchedule) {
    this.recurringSchedule = recurringSchedule;
  }

  @Override
  public String toString() {
    return "Schedules [timeZone="
        + timeZone
        + ", specificDate="
        + specificDate
        + ", recurringSchedule="
        + recurringSchedule
        + "]";
  }
}
