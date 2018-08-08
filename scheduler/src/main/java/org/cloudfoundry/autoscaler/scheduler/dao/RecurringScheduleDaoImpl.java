package org.cloudfoundry.autoscaler.scheduler.dao;

import java.util.List;

import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.error.DatabaseValidationException;
import org.springframework.stereotype.Repository;
import org.springframework.transaction.annotation.Transactional;

@Repository("recurringScheduleDao")
public class RecurringScheduleDaoImpl extends GenericDaoImpl<RecurringScheduleEntity> implements RecurringScheduleDao {

	@Override
	public List<RecurringScheduleEntity> findAllRecurringSchedulesByAppId(String appId) {
		try {
			return entityManager.createNamedQuery(RecurringScheduleEntity.query_recurringSchedulesByAppId,
					RecurringScheduleEntity.class).setParameter("appId", appId).getResultList();

		} catch (Exception e) {
			throw new DatabaseValidationException("Find All recurring schedules by app id failed", e);
		}
	}

	@Override
	@Transactional(readOnly = true)
	public List getDistinctAppIdAndGuidList() {
		try {
			return entityManager.createNamedQuery(RecurringScheduleEntity.query_findDistinctAppIdAndGuidFromRecurringSchedule).getResultList();
		} catch (Exception e) {
			throw new DatabaseValidationException("Find All recurring schedules failed", e);
		}
	}

}
