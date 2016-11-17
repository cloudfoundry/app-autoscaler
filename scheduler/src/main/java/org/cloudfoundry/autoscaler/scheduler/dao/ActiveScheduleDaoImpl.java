package org.cloudfoundry.autoscaler.scheduler.dao;

import java.util.List;

import javax.sql.DataSource;

import org.cloudfoundry.autoscaler.scheduler.entity.ActiveScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.util.error.DatabaseValidationException;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.dao.DataAccessException;
import org.springframework.dao.EmptyResultDataAccessException;
import org.springframework.jdbc.core.support.JdbcDaoSupport;
import org.springframework.stereotype.Repository;

@Repository("activeScheduleDao")
public class ActiveScheduleDaoImpl extends JdbcDaoSupport implements ActiveScheduleDao {

	private static final String TABLE_NAME = "app_scaling_active_schedule";

	private static final String SELECT_SQL = "SELECT * FROM " + TABLE_NAME + " WHERE id=?";

	private static final String INSERT_SQL = "INSERT INTO " + TABLE_NAME
			+ "(id, app_id, instance_min_count, instance_max_count, initial_min_instance_count) "
			+ "VALUES (?, ?, ?, ?, ?)";

	private static final String DELETE_SQL = "DELETE FROM " + TABLE_NAME + " WHERE id=?";

	private static final String DELETE_ALL_FOR_APPID_SQL = "DELETE FROM " + TABLE_NAME + " WHERE app_id=?";

	@Autowired
	private void setupDataSource(DataSource dataSource) {
		setDataSource(dataSource);
	}

	@Override
	public ActiveScheduleEntity find(Long id) {
		try {
			return getJdbcTemplate().queryForObject(SELECT_SQL, new Object[] { id }, new ActiveScheduleEntity());
		} catch (EmptyResultDataAccessException ex) {
			return null;
		} catch (DataAccessException e) {
			throw new DatabaseValidationException("Find failed", e);
		}
	}

	@Override
	public void create(ActiveScheduleEntity activeScheduleEntity) {
		Object[] objects = new Object[] { activeScheduleEntity.getId(), activeScheduleEntity.getAppId(),
				activeScheduleEntity.getInstanceMinCount(), activeScheduleEntity.getInstanceMaxCount(),
				activeScheduleEntity.getInitialMinInstanceCount()};
		try {
			getJdbcTemplate().update(INSERT_SQL, objects);
		} catch (DataAccessException e) {
			throw new DatabaseValidationException("Create failed", e);
		}
	}

	@Override
	public int delete(Long id) {
		try {
			return getJdbcTemplate().update(DELETE_SQL, id);

		} catch (DataAccessException e) {
			throw new DatabaseValidationException("Delete failed", e);
		}
	}

	@Override
	public void deleteAllActiveSchedulesByAppId(String appId) {
		try {
			getJdbcTemplate().update(DELETE_ALL_FOR_APPID_SQL, new Object[] { appId });
		} catch (DataAccessException e) {
			throw new DatabaseValidationException("Delete All active schedules for Application Id:" + appId + " failed",
					e);
		}
	}
}
