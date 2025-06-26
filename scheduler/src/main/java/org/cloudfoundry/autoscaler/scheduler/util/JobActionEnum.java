package org.cloudfoundry.autoscaler.scheduler.util;

public enum JobActionEnum {
  START("Start Scaling Job", "_start"),
  END("End Scaling Job", "_end");

  private String description;
  private String jobIdSuffix;

  JobActionEnum(String description, String jobIdSuffix) {
    this.description = description;
    this.jobIdSuffix = jobIdSuffix;
  }

  public String getDescription() {
    return description;
  }

  public String getJobIdSuffix() {
    return jobIdSuffix;
  }
}
