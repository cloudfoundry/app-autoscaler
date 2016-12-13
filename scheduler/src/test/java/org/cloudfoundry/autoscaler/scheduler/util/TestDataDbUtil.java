package org.cloudfoundry.autoscaler.scheduler.util;

import java.util.ArrayList;
import java.util.List;
import java.util.Set;

import javax.sql.DataSource;

import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.quartz.JobKey;
import org.quartz.Scheduler;
import org.quartz.SchedulerException;
import org.quartz.impl.matchers.GroupMatcher;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Component;

@Component
public class TestDataDbUtil {

	@Autowired
	private DataSource dataSource;

	public void cleanupData() {
		removeAllActiveSchedules();
		removeAllSpecificDateSchedules();
		removeAllRecurringSchedules();
	}

	public void cleanupData(Scheduler scheduler) throws SchedulerException {
		removeAllActiveSchedules();
		removeAllSpecificDateSchedules();
		removeAllRecurringSchedules();
		cleanScheduler(scheduler);
	}

	public Long getCurrentSequenceSchedulerId() {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);

		return jdbcTemplate.queryForObject("SELECT last_value from schedule_id_sequence;", Long.class);
	}

	private Long numberingScheduleId() {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
		return jdbcTemplate.queryForObject("SELECT nextval('schedule_id_sequence');", Long.class);
	}

	public int getNumberOfSpecificDateSchedules() {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
		return jdbcTemplate.queryForObject("SELECT COUNT(1) FROM app_scaling_specific_date_schedule", Integer.class);
	}

	public int getNumberOfRecurringSchedules() {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
		return jdbcTemplate.queryForObject("SELECT COUNT(1) FROM app_scaling_recurring_schedule", Integer.class);
	}

	public int getNumberOfSpecificDateSchedulesByAppId(String appId) {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
		return jdbcTemplate.queryForObject("SELECT COUNT(1) FROM app_scaling_specific_date_schedule WHERE app_id=?",
				new Object[] { appId }, Integer.class);
	}

	public int getNumberOfRecurringSchedulesByAppId(String appId) {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
		return jdbcTemplate.queryForObject("SELECT COUNT(1) FROM app_scaling_recurring_schedule WHERE app_id=?",
				new Object[] { appId }, Integer.class);
	}

	public long getNumberOfActiveSchedules() {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
		return jdbcTemplate.queryForObject("SELECT COUNT(1) FROM app_scaling_active_schedule", Long.class);
	}

	public long getNumberOfActiveSchedulesByAppId(String appId) {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
		return jdbcTemplate.queryForObject("SELECT COUNT(1) FROM app_scaling_active_schedule WHERE app_id=?",
				new Object[] { appId }, Long.class);
	}

	public long getNumberOfActiveSchedulesByScheduleId(Long scheduleId) {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
		return jdbcTemplate.queryForObject("SELECT COUNT(1) FROM app_scaling_active_schedule WHERE id=?",
				new Object[] { scheduleId }, Long.class);
	}

	public void insertSpecificDateSchedule(List<SpecificDateScheduleEntity> entities) {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
		for (SpecificDateScheduleEntity entity : entities) {
			Long scheduleId = numberingScheduleId();
			Object[] objects = new Object[] { scheduleId, entity.getAppId(), entity.getTimeZone(),
					entity.getStartDateTime(), entity.getEndDateTime(), entity.getInstanceMinCount(),
					entity.getInstanceMaxCount(), entity.getDefaultInstanceMinCount(),
					entity.getDefaultInstanceMaxCount(), entity.getInitialMinInstanceCount() };

			jdbcTemplate.update("INSERT INTO app_scaling_specific_date_schedule "
					+ "(schedule_id, app_id, timezone, start_date_time, end_date_time, instance_min_count, instance_max_count, default_instance_min_count, default_instance_max_count, initial_min_instance_count) "
					+ "VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", objects);
		}
	}

	public void insertRecurringSchedule(List<RecurringScheduleEntity> entities) {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);

		for (RecurringScheduleEntity entity : entities) {
			Long scheduleId = numberingScheduleId();

			Object[] objects = new Object[] { scheduleId, entity.getAppId(), entity.getTimeZone(),
					entity.getDefaultInstanceMinCount(), entity.getDefaultInstanceMaxCount(),
					entity.getInstanceMinCount(), entity.getInstanceMaxCount(), entity.getInitialMinInstanceCount(),
					entity.getStartDate(), entity.getEndDate(), entity.getStartTime(), entity.getEndTime(),
					convertArrayToBits(entity.getDaysOfWeek()), convertArrayToBits(entity.getDaysOfMonth()) };

			jdbcTemplate.update("INSERT INTO app_scaling_recurring_schedule "
					+ "( schedule_id, app_id, timezone, default_instance_min_count, default_instance_max_count, instance_min_count, instance_max_count, initial_min_instance_count, start_date, end_date, start_time, end_time, days_of_week, days_of_month) "
					+ "VALUES ( ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", objects);
		}
	}

	public void insertActiveSchedule(String appId, Long scheduleId, int instanceMinCount, int instanceMaxCount,
			int initialMinInstanceCount, Long startJobIdentifier) {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);

		Object[] objects = new Object[] { scheduleId, appId, startJobIdentifier, instanceMinCount, instanceMaxCount,
				initialMinInstanceCount };

		jdbcTemplate.update("INSERT INTO app_scaling_active_schedule "
				+ "(id, app_id, start_job_identifier, instance_min_count, instance_max_count, initial_min_instance_count) "
				+ "VALUES (?, ?, ?, ?, ?, ?)", objects);
	}

	private void removeAllActiveSchedules() {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);

		jdbcTemplate.update("DELETE FROM app_scaling_active_schedule;");
	}

	private void removeAllSpecificDateSchedules() {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);

		jdbcTemplate.update("DELETE FROM app_scaling_specific_date_schedule");
	}

	private void removeAllRecurringSchedules() {
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);

		jdbcTemplate.update("DELETE FROM app_scaling_recurring_schedule");
	}

	private void cleanScheduler(Scheduler scheduler) throws SchedulerException {
		scheduler.clear();

		Set<JobKey> jobKeys = scheduler.getJobKeys(GroupMatcher.anyGroup());
		scheduler.deleteJobs(new ArrayList<>(jobKeys));
	}

	private int convertArrayToBits(int[] values) {
		int bits = 0;

		if (values == null) {
			bits = 0;
		} else {
			for (int value : values) {
				bits |= 1 << (value - 1);
			}

		}
		return bits;
	}

}
