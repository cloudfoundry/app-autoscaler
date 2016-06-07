package org.cloudfoundry.autoscaler.scheduler.dao;

import java.util.List;

import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.springframework.stereotype.Repository;

/**
 *
 * @author Fujitsu
 *
 */
@Repository("scheduleDao")
public class ScheduleDaoImpl extends GenericDaoImpl<ScheduleEntity> implements ScheduleDao {

	public ScheduleDaoImpl() {
		super(ScheduleEntity.class);
	}

	@Override
	public List<ScheduleEntity> findAllSchedulesByAppId(String appId) {
		@SuppressWarnings("unchecked")
		List<ScheduleEntity> scheduleEntities = entityManager.createNamedQuery(ScheduleEntity.query_schedulesByAppId)
				.setParameter("appId", appId).getResultList();

		return scheduleEntities;

	}

}
