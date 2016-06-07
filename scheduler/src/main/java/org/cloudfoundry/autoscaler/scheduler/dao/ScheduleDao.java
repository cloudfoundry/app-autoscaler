package org.cloudfoundry.autoscaler.scheduler.dao;

import java.util.List;

import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;

/**
 * @author Fujitsu
 *
 */
public interface ScheduleDao extends GenericDao<ScheduleEntity> {

	public List<ScheduleEntity> findAllSchedulesByAppId(String appId);

}
