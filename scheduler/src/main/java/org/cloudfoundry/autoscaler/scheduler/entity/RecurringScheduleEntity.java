package org.cloudfoundry.autoscaler.scheduler.entity;

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
import java.time.LocalDate;
import java.time.LocalTime;
import java.util.Arrays;
import org.cloudfoundry.autoscaler.scheduler.util.DateDeserializer;
import org.cloudfoundry.autoscaler.scheduler.util.DateHelper;
import org.cloudfoundry.autoscaler.scheduler.util.DateSerializer;
import org.cloudfoundry.autoscaler.scheduler.util.TimeDeserializer;
import org.cloudfoundry.autoscaler.scheduler.util.TimeSerializer;
import org.hibernate.annotations.Type;

@ApiModel
@Entity
@Table(name = "app_scaling_recurring_schedule")
@NamedQuery(
    name = RecurringScheduleEntity.query_recurringSchedulesByAppId,
    query = RecurringScheduleEntity.jpql_recurringSchedulesByAppId)
@NamedQuery(
    name = RecurringScheduleEntity.query_findDistinctAppIdAndGuidFromRecurringSchedule,
    query = RecurringScheduleEntity.jpql_findDistinctAppIdAndGuidFromRecurringSchedule)
public class RecurringScheduleEntity extends ScheduleEntity {

  @ApiModelProperty(
      example = DateHelper.TIME_FORMAT,
      dataType = "java.lang.String",
      required = true,
      position = 3)
  @JsonDeserialize(using = TimeDeserializer.class)
  @JsonSerialize(using = TimeSerializer.class)
  @NotNull
  @Column(name = "start_time")
  @JsonProperty(value = "start_time")
  private LocalTime startTime;

  @ApiModelProperty(
      example = DateHelper.TIME_FORMAT,
      dataType = "java.lang.String",
      required = true,
      position = 4)
  @JsonDeserialize(using = TimeDeserializer.class)
  @JsonSerialize(using = TimeSerializer.class)
  @NotNull
  @Column(name = "end_time")
  @JsonProperty(value = "end_time")
  private LocalTime endTime;

  @ApiModelProperty(example = DateHelper.DATE_FORMAT, position = 1)
  @JsonDeserialize(using = DateDeserializer.class)
  @JsonSerialize(using = DateSerializer.class)
  @Column(name = "start_date")
  @JsonProperty(value = "start_date")
  private LocalDate startDate;

  @ApiModelProperty(example = DateHelper.DATE_FORMAT, position = 2)
  @JsonDeserialize(using = DateDeserializer.class)
  @JsonSerialize(using = DateSerializer.class)
  @Column(name = "end_date")
  @JsonProperty(value = "end_date")
  private LocalDate endDate;

  @ApiModelProperty(example = "[2, 3, 4, 5]", hidden = true)
  @Type(value = org.cloudfoundry.autoscaler.scheduler.entity.BitsetUserType.class)
  @Column(name = "days_of_week")
  @JsonProperty(value = "days_of_week")
  private int[] daysOfWeek;

  @ApiModelProperty(example = "[10, 20, 25]", position = 6)
  @Type(value = org.cloudfoundry.autoscaler.scheduler.entity.BitsetUserType.class)
  @Column(name = "days_of_month")
  @JsonProperty(value = "days_of_month")
  private int[] daysOfMonth;

  public int[] getDaysOfWeek() {
    return daysOfWeek;
  }

  public void setDaysOfWeek(int[] daysOfWeek) {
    this.daysOfWeek = daysOfWeek;
  }

  public int[] getDaysOfMonth() {
    return daysOfMonth;
  }

  public void setDaysOfMonth(int[] daysOfMonth) {
    this.daysOfMonth = daysOfMonth;
  }

  @JsonProperty("start_time")
  public LocalTime getStartTime() {
    return startTime;
  }

  @JsonProperty("start_time")
  public void setStartTime(LocalTime startTime) {
    this.startTime = startTime;
  }

  public LocalTime getEndTime() {
    return endTime;
  }

  public void setEndTime(LocalTime endTime) {
    this.endTime = endTime;
  }

  public LocalDate getStartDate() {
    return startDate;
  }

  public void setStartDate(LocalDate startDate) {
    this.startDate = startDate;
  }

  public LocalDate getEndDate() {
    return endDate;
  }

  public void setEndDate(LocalDate endDate) {
    this.endDate = endDate;
  }

  public static final String query_recurringSchedulesByAppId =
      "RecurringScheduleEntity.schedulesByAppId";
  static final String jpql_recurringSchedulesByAppId =
      "FROM RecurringScheduleEntity WHERE appId = :appId";
  public static final String query_findDistinctAppIdAndGuidFromRecurringSchedule =
      "RecurringScheduleEntity.findDistinctAppIdAndGuid";
  static final String jpql_findDistinctAppIdAndGuidFromRecurringSchedule =
      "SELECT DISTINCT appId, guid FROM RecurringScheduleEntity";

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

    RecurringScheduleEntity that = (RecurringScheduleEntity) o;
    if (!startTime.equals(that.startTime)) {
      return false;
    }
    if (!endTime.equals(that.endTime)) {
      return false;
    }
    if (startDate != null ? !startDate.equals(that.startDate) : that.startDate != null) {
      return false;
    }
    if (endDate != null ? !endDate.equals(that.endDate) : that.endDate != null) {
      return false;
    }
    if (!Arrays.equals(daysOfWeek, that.daysOfWeek)) {
      return false;
    }
    return Arrays.equals(daysOfMonth, that.daysOfMonth);
  }

  @Override
  public int hashCode() {
    int result = super.hashCode();
    result = 31 * result + startTime.hashCode();
    result = 31 * result + endTime.hashCode();
    result = 31 * result + (startDate != null ? startDate.hashCode() : 0);
    result = 31 * result + (endDate != null ? endDate.hashCode() : 0);
    result = 31 * result + Arrays.hashCode(daysOfWeek);
    result = 31 * result + Arrays.hashCode(daysOfMonth);
    return result;
  }

  @Override
  public String toString() {
    return super.toString()
        + ", RecurringScheduleEntity [startTime="
        + startTime
        + ", endTime="
        + endTime
        + ", startDate="
        + startDate
        + ", endDate="
        + endDate
        + ", dayOfWeek="
        + Arrays.toString(daysOfWeek)
        + ", dayOfMonth="
        + Arrays.toString(daysOfMonth)
        + "]";
  }
}
