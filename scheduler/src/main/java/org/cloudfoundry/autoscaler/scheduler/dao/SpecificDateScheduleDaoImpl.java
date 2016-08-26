package org.cloudfoundry.autoscaler.scheduler.dao;

import java.util.List;

import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.error.DatabaseValidationException;
import org.springframework.stereotype.Repository;

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
	 * @see org.cloudfoundry.autoscaler.scheduler.dao.SpecificDateScheduleDao#findAllSpecificDateSchedulesByAppId(java.lang.String)
	 */
	@Override
	public List<SpecificDateScheduleEntity> findAllSpecificDateSchedulesByAppId(String appId) {
		try {
			return entityManager
					.createNamedQuery(SpecificDateScheduleEntity.query_specificDateSchedulesByAppId,
							SpecificDateScheduleEntity.class)
					.setParameter("app_id", appId).getResultList();
			
		} catch(Exception exception){
			
			throw new DatabaseValidationException("Find All specific date schedules failed", exception);
		}
	}

}
