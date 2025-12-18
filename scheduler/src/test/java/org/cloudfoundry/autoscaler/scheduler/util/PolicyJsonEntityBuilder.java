package org.cloudfoundry.autoscaler.scheduler.util;

import tools.jackson.databind.ObjectMapper;
import org.cloudfoundry.autoscaler.scheduler.entity.PolicyJsonEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationSchedules;
import tools.jackson.core.JacksonException;

public class PolicyJsonEntityBuilder {

  private PolicyJsonEntity policyJson;

  public PolicyJsonEntityBuilder(String appId, String guid, ApplicationSchedules schedules)
      throws JacksonException {
    this.policyJson = new PolicyJsonEntity();
    this.policyJson.setAppId(appId);
    this.policyJson.setGuid(guid);
    ObjectMapper mapper = new ObjectMapper();
    this.policyJson.setPolicyJson(mapper.writeValueAsString(schedules));
  }

  public PolicyJsonEntity build() {
    return this.policyJson;
  }
}
