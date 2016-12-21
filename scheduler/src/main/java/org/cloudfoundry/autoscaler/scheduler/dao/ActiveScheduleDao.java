package org.cloudfoundry.autoscaler.scheduler.dao;

import org.cloudfoundry.autoscaler.scheduler.entity.ActiveScheduleEntity;

import java.util.List;

public interface ActiveScheduleDao {

	ActiveScheduleEntity find(Long id);

	void create(ActiveScheduleEntity activeScheduleEntity);

	int delete(Long id, Long startJobIdentifier);

	int deleteActiveSchedulesByAppId(String appId);

	List<ActiveScheduleEntity> findByAppId(String appId);
}
