package org.cloudfoundry.autoscaler.scheduler.rest.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class SynchronizeResult {

  @JsonProperty(value = "createCount")
  private Integer createCount;

  @JsonProperty(value = "updateCount")
  private Integer updateCount;

  @JsonProperty(value = "deleteCount")
  private Integer deleteCount;

  public SynchronizeResult() {}

  public SynchronizeResult(Integer createCount, Integer updateCount, Integer deleteCount) {
    super();
    this.createCount = createCount;
    this.updateCount = updateCount;
    this.deleteCount = deleteCount;
  }

  public Integer getCreateCount() {
    return createCount;
  }

  public void setCreateCount(Integer createCount) {
    this.createCount = createCount;
  }

  public Integer getUpdateCount() {
    return updateCount;
  }

  public void setUpdateCount(Integer updateCount) {
    this.updateCount = updateCount;
  }

  public Integer getDeleteCount() {
    return deleteCount;
  }

  public void setDeleteCount(Integer deleteCount) {
    this.deleteCount = deleteCount;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }

    SynchronizeResult that = (SynchronizeResult) o;
    if (!createCount.equals(that.createCount)) {
      return false;
    }
    if (!updateCount.equals(that.updateCount)) {
      return false;
    }
    if (!deleteCount.equals(that.deleteCount)) {
      return false;
    }
    return true;
  }
}
