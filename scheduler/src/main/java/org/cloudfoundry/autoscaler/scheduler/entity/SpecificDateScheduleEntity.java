package org.cloudfoundry.autoscaler.scheduler.entity;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;
import com.fasterxml.jackson.databind.annotation.JsonSerialize;
import io.swagger.annotations.ApiModel;
import io.swagger.annotations.ApiModelProperty;
import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.NamedQuery;
import jakarta.persistence.Table;
import jakarta.validation.constraints.NotNull;
import java.time.LocalDateTime;
import org.cloudfoundry.autoscaler.scheduler.util.DateHelper;
import org.cloudfoundry.autoscaler.scheduler.util.DateTimeDeserializer;
import org.cloudfoundry.autoscaler.scheduler.util.DateTimeSerializer;

@ApiModel
@Entity
@Table(name = "app_scaling_specific_date_schedule")
@NamedQuery(
    name = SpecificDateScheduleEntity.query_specificDateSchedulesByAppId,
    query = SpecificDateScheduleEntity.jpql_specificDateSchedulesByAppId)
@NamedQuery(
    name = SpecificDateScheduleEntity.query_findDistinctAppIdAndGuidFromSpecificDateSchedule,
    query = SpecificDateScheduleEntity.jpql_findDistinctAppIdAndGuidFromSpecificDateSchedule)
public class SpecificDateScheduleEntity extends ScheduleEntity {

  @ApiModelProperty(example = DateHelper.DATE_TIME_FORMAT, required = true, position = 1)
  @JsonFormat(pattern = DateHelper.DATE_TIME_FORMAT)
  @JsonDeserialize(using = DateTimeDeserializer.class)
  @JsonSerialize(using = DateTimeSerializer.class)
  @NotNull
  @Column(name = "start_date_time")
  @JsonProperty("start_date_time")
  private LocalDateTime startDateTime;

  @ApiModelProperty(example = DateHelper.DATE_TIME_FORMAT, required = true, position = 2)
  @JsonFormat(pattern = DateHelper.DATE_TIME_FORMAT)
  @JsonDeserialize(using = DateTimeDeserializer.class)
  @JsonSerialize(using = DateTimeSerializer.class)
  @NotNull
  @Column(name = "end_date_time")
  @JsonProperty("end_date_time")
  private LocalDateTime endDateTime;

  public LocalDateTime getStartDateTime() {
    return startDateTime;
  }

  public void setStartDateTime(LocalDateTime startDateTime) {
    this.startDateTime = startDateTime;
  }

  public LocalDateTime getEndDateTime() {
    return endDateTime;
  }

  public void setEndDateTime(LocalDateTime endDateTime) {
    this.endDateTime = endDateTime;
  }

  public static final String query_specificDateSchedulesByAppId =
      "SpecificDateScheduleEntity.schedulesByAppId";
  static final String jpql_specificDateSchedulesByAppId =
      " FROM SpecificDateScheduleEntity WHERE appId = :appId";

  public static final String query_findDistinctAppIdAndGuidFromSpecificDateSchedule =
      "SpecificDateScheduleEntity.findDistinctAppIdAndGuid";
  static final String jpql_findDistinctAppIdAndGuidFromSpecificDateSchedule =
      "SELECT DISTINCT appId,guid FROM SpecificDateScheduleEntity";

  @Override
  public boolean equals(Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    if (!super.equals(o)) {
      return false;
    }
    SpecificDateScheduleEntity that = (SpecificDateScheduleEntity) o;

    if (!startDateTime.equals(that.startDateTime)) {
      return false;
    }
    return endDateTime.equals(that.endDateTime);
  }

  @Override
  public int hashCode() {
    int result = super.hashCode();
    result = 31 * result + startDateTime.hashCode();
    result = 31 * result + endDateTime.hashCode();
    return result;
  }

  @Override
  public String toString() {
    return super.toString()
        + ", SpecificDateScheduleEntity [startDateTime="
        + startDateTime
        + ", endDateTime="
        + endDateTime
        + "]";
  }
}
