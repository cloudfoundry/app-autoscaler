package org.cloudfoundry.autoscaler.scheduler.dao;

import java.util.List;
import org.cloudfoundry.autoscaler.scheduler.entity.PolicyJsonEntity;

public interface PolicyJsonDao {

  public List<PolicyJsonEntity> getAllPolicies();
}
