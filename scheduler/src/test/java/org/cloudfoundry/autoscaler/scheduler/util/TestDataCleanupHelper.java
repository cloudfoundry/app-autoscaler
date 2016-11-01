package org.cloudfoundry.autoscaler.scheduler.util;

import java.util.ArrayList;
import java.util.Set;

import javax.sql.DataSource;

import org.quartz.JobKey;
import org.quartz.Scheduler;
import org.quartz.SchedulerException;
import org.quartz.impl.matchers.GroupMatcher;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Component;

@Component
public class TestDataCleanupHelper {

	@Autowired
	private DataSource dataSource;


	public void cleanupData(){
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

	public Long getCurrentSequenceSchedulerId(){
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);

		return jdbcTemplate.queryForObject("SELECT last_value from schedule_id_sequence;", Long.class);
	}

	private void removeAllActiveSchedules(){
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);

		jdbcTemplate.update("DELETE FROM app_scaling_active_schedule;");
	}


	private  void removeAllSpecificDateSchedules(){
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);

		jdbcTemplate.update("DELETE FROM app_scaling_specific_date_schedule");
	}

	private  void removeAllRecurringSchedules(){
		JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);

		jdbcTemplate.update("DELETE FROM app_scaling_recurring_schedule");
	}

	private void cleanScheduler(Scheduler scheduler)throws SchedulerException {
		scheduler.clear();

		Set<JobKey> jobKeys = scheduler.getJobKeys(GroupMatcher.anyGroup());
		scheduler.deleteJobs(new ArrayList<>(jobKeys));
	}
}
