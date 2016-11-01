package org.cloudfoundry.autoscaler.scheduler.dao;

import org.cloudfoundry.autoscaler.scheduler.entity.ActiveScheduleEntity;

public interface ActiveScheduleDao {

	ActiveScheduleEntity find(Long id);

	void create(ActiveScheduleEntity activeScheduleEntity);

	int delete(Long id);
}
