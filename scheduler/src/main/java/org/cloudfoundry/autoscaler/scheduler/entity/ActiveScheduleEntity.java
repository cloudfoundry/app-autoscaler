package org.cloudfoundry.autoscaler.scheduler.entity;

import com.fasterxml.jackson.annotation.JsonIgnore;
import com.fasterxml.jackson.annotation.JsonProperty;
import io.swagger.annotations.ApiModel;
import io.swagger.annotations.ApiModelProperty;
import java.sql.ResultSet;
import java.sql.SQLException;
import org.springframework.jdbc.core.RowMapper;

@ApiModel
public class ActiveScheduleEntity implements RowMapper<ActiveScheduleEntity> {

  @JsonIgnore private Long id;

  @JsonIgnore
  @JsonProperty(value = "app_id")
  private String appId;

  @JsonIgnore private Long startJobIdentifier;

  @ApiModelProperty(required = true, position = 1)
  @JsonProperty(value = "instance_min_count")
  private Integer instanceMinCount;

  @ApiModelProperty(required = true, position = 2)
  @JsonProperty(value = "instance_max_count")
  private Integer instanceMaxCount;

  @ApiModelProperty(position = 3)
  @JsonProperty(value = "initial_min_instance_count")
  private Integer initialMinInstanceCount;

  public Long getId() {
    return id;
  }

  public void setId(Long id) {
    this.id = id;
  }

  public String getAppId() {
    return appId;
  }

  public void setAppId(String appId) {
    this.appId = appId;
  }

  public Long getStartJobIdentifier() {
    return startJobIdentifier;
  }

  public void setStartJobIdentifier(Long startJobIdentifier) {
    this.startJobIdentifier = startJobIdentifier;
  }

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

  public Integer getInitialMinInstanceCount() {
    return initialMinInstanceCount;
  }

  public void setInitialMinInstanceCount(Integer initialMinInstanceCount) {
    this.initialMinInstanceCount = initialMinInstanceCount;
  }

  public ActiveScheduleEntity mapRow(ResultSet rs, int rowNum) throws SQLException {
    ActiveScheduleEntity activeScheduleEntity = new ActiveScheduleEntity();
    activeScheduleEntity.setId(rs.getLong("id"));
    activeScheduleEntity.setAppId(rs.getString("app_id"));
    activeScheduleEntity.setInstanceMinCount(rs.getInt("instance_min_count"));
    activeScheduleEntity.setInstanceMaxCount(rs.getInt("instance_max_count"));
    activeScheduleEntity.setStartJobIdentifier(rs.getLong("start_job_identifier"));

    int initialMinInstanceCount = rs.getInt("initial_min_instance_count");
    activeScheduleEntity.setInitialMinInstanceCount(rs.wasNull() ? null : initialMinInstanceCount);

    return activeScheduleEntity;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    ActiveScheduleEntity that = (ActiveScheduleEntity) o;

    if (!id.equals(that.id)) {
      return false;
    }
    if (!appId.equals(that.appId)) {
      return false;
    }
    if (startJobIdentifier != null
        ? !startJobIdentifier.equals(that.startJobIdentifier)
        : that.startJobIdentifier != null) {
      return false;
    }
    if (!instanceMinCount.equals(that.instanceMinCount)) {
      return false;
    }
    if (!instanceMaxCount.equals(that.instanceMaxCount)) {
      return false;
    }
    return initialMinInstanceCount != null
        ? initialMinInstanceCount.equals(that.initialMinInstanceCount)
        : that.initialMinInstanceCount == null;
  }

  @Override
  public int hashCode() {
    int result = id.hashCode();
    result = 31 * result + appId.hashCode();
    result = 31 * result + (startJobIdentifier != null ? startJobIdentifier.hashCode() : 0);
    result = 31 * result + instanceMinCount.hashCode();
    result = 31 * result + instanceMaxCount.hashCode();
    result =
        31 * result + (initialMinInstanceCount != null ? initialMinInstanceCount.hashCode() : 0);
    return result;
  }

  @Override
  public String toString() {
    return "ActiveScheduleEntity{"
        + "id="
        + id
        + ", appId='"
        + appId
        + '\''
        + ", startJobIdentifier="
        + startJobIdentifier
        + ", instanceMinCount="
        + instanceMinCount
        + ", instanceMaxCount="
        + instanceMaxCount
        + ", initialMinInstanceCount="
        + initialMinInstanceCount
        + '}';
  }
}
