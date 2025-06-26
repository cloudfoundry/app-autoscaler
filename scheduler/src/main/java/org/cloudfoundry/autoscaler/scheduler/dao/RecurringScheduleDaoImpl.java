package org.cloudfoundry.autoscaler.scheduler.dao;

import java.util.HashMap;
import java.util.List;
import java.util.Map;
import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.error.DatabaseValidationException;
import org.springframework.stereotype.Repository;
import org.springframework.transaction.annotation.Transactional;

@Repository("recurringScheduleDao")
public class RecurringScheduleDaoImpl extends GenericDaoImpl<RecurringScheduleEntity>
    implements RecurringScheduleDao {

  @Override
  public List<RecurringScheduleEntity> findAllRecurringSchedulesByAppId(String appId) {
    try {
      return entityManager
          .createNamedQuery(
              RecurringScheduleEntity.query_recurringSchedulesByAppId,
              RecurringScheduleEntity.class)
          .setParameter("appId", appId)
          .getResultList();

    } catch (Exception e) {
      throw new DatabaseValidationException("Find All recurring schedules by app id failed", e);
    }
  }

  @Override
  @Transactional(readOnly = true)
  public Map<String, String> getDistinctAppIdAndGuidList() {
    try {
      List<Object[]> res =
          entityManager
              .createNamedQuery(
                  RecurringScheduleEntity.query_findDistinctAppIdAndGuidFromRecurringSchedule,
                  Object[].class)
              .getResultList();

      Map<String, String> result = new HashMap<>(res.size());

      for (Object[] r : res) {
        result.put((String) (r[0]), (String) (r[1]));
      }

      return result;
    } catch (Exception e) {
      throw new DatabaseValidationException("Find All recurring schedules failed", e);
    }
  }
}
