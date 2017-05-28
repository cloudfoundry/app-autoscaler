package org.cloudfoundry.autoscaler.scheduler.dao;

import java.util.List;

import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.error.DatabaseValidationException;
import org.springframework.stereotype.Repository;
import org.springframework.transaction.annotation.Transactional;

/**
 *
 * 
 *
 */
@Repository("specificDateScheduleDao")
public class SpecificDateScheduleDaoImpl extends GenericDaoImpl<SpecificDateScheduleEntity>
		implements SpecificDateScheduleDao {

	public SpecificDateScheduleDaoImpl() {
		super(SpecificDateScheduleEntity.class);
	}

	/*
	 * Gets all the specific date schedules for the specified application Id
	 * 
	 * @see org.cloudfoundry.autoscaler.scheduler.dao.SpecificDateScheduleDao#
	 * findAllSpecificDateSchedulesByAppId(java.lang.String)
	 */
	@Override
	public List<SpecificDateScheduleEntity> findAllSpecificDateSchedulesByAppId(String appId) {
		try {
			return entityManager.createNamedQuery(SpecificDateScheduleEntity.query_specificDateSchedulesByAppId,
					SpecificDateScheduleEntity.class).setParameter("appId", appId).getResultList();

		} catch (Exception e) {

			throw new DatabaseValidationException("Find All specific date schedules by app id failed", e);
		}
	}

	@Override
	@Transactional(readOnly = true)
	public List getDistinctAppIdAndGuidList() {
		try {
			return entityManager.createNamedQuery(SpecificDateScheduleEntity.query_findDistinctAppIdAndGuidFromSpecificDateSchedule).getResultList();

		} catch (Exception e) {

			throw new DatabaseValidationException("Find All specific date schedules failed", e);
		}
	}

}
