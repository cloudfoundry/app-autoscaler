package org.cloudfoundry.autoscaler.scheduler.util;

public class ScalingEngineUtil {

  public static String getScalingEngineActiveSchedulePath(
      String scalingEngineUrl, String appId, Long scheduleId) {

    return scalingEngineUrl + "/v1/apps/" + appId + "/active_schedules/" + scheduleId;
  }
}
