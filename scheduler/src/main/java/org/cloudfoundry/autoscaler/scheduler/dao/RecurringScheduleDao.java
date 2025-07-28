package org.cloudfoundry.autoscaler.scheduler.dao;

import java.util.List;
import java.util.Map;
import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;

public interface RecurringScheduleDao extends GenericDao<RecurringScheduleEntity> {

  public List<RecurringScheduleEntity> findAllRecurringSchedulesByAppId(String appId);

  public Map<String, String> getDistinctAppIdAndGuidList();
}
