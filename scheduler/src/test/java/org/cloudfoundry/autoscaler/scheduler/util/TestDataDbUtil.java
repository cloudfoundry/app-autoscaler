package org.cloudfoundry.autoscaler.scheduler.util;

import jakarta.annotation.Resource;
import java.sql.Time;
import java.sql.Timestamp;
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
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Transactional;

@Component
public class TestDataDbUtil {

  @Resource(name = "dataSource")
  private DataSource dataSource;

  @Resource(name = "dataSource")
  private DataSource policyDbDataSource;

  private DatabaseType databaseType = null;

  public void cleanupData() {
    removeAllActiveSchedules();
    removeAllSpecificDateSchedules();
    removeAllRecurringSchedules();
    removeAllPolicyJson();
  }

  public void cleanupData(Scheduler scheduler) throws SchedulerException {
    removeAllActiveSchedules();
    removeAllSpecificDateSchedules();
    removeAllRecurringSchedules();
    cleanScheduler(scheduler);
  }

  public Long getCurrentSpecificDateSchedulerId() throws Exception {
    JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);

    String sql = "SELECT MAX(schedule_id) FROM app_scaling_specific_date_schedule;";
    return jdbcTemplate.queryForObject(sql, Long.class);
  }

  public Long getCurrentRecurringSchedulerId() throws Exception {
    JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
    String sql = "SELECT MAX(schedule_id) FROM app_scaling_recurring_schedule;";
    return jdbcTemplate.queryForObject(sql, Long.class);
  }

  public int getNumberOfSpecificDateSchedules() {
    JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
    return jdbcTemplate.queryForObject(
        "SELECT COUNT(1) FROM app_scaling_specific_date_schedule", Integer.class);
  }

  public int getNumberOfRecurringSchedules() {
    JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
    return jdbcTemplate.queryForObject(
        "SELECT COUNT(1) FROM app_scaling_recurring_schedule", Integer.class);
  }

  public int getNumberOfSpecificDateSchedulesByAppId(String appId) {
    JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
    return jdbcTemplate.queryForObject(
        "SELECT COUNT(1) FROM app_scaling_specific_date_schedule WHERE app_id=?",
        Integer.class,
        new Object[] {appId});
  }

  public int getNumberOfRecurringSchedulesByAppId(String appId) {
    JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
    return jdbcTemplate.queryForObject(
        "SELECT COUNT(1) FROM app_scaling_recurring_schedule WHERE app_id=?",
        Integer.class,
        new Object[] {appId});
  }

  public long getNumberOfActiveSchedules() {
    JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
    return jdbcTemplate.queryForObject(
        "SELECT COUNT(1) FROM app_scaling_active_schedule", Long.class);
  }

  public long getNumberOfActiveSchedulesByAppId(String appId) {
    JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
    return jdbcTemplate.queryForObject(
        "SELECT COUNT(1) FROM app_scaling_active_schedule WHERE app_id=?",
        Long.class,
        new Object[] {appId});
  }

  public long getNumberOfActiveSchedulesByScheduleId(Long scheduleId) {
    JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
    return jdbcTemplate.queryForObject(
        "SELECT COUNT(1) FROM app_scaling_active_schedule WHERE id=?",
        Long.class,
        new Object[] {scheduleId});
  }

  public void insertSpecificDateSchedule(List<SpecificDateScheduleEntity> entities) {
    JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);
    for (SpecificDateScheduleEntity entity : entities) {
      Object[] objects =
          new Object[] {
            entity.getAppId(),
            entity.getTimeZone(),
            Timestamp.valueOf(entity.getStartDateTime()),
            Timestamp.valueOf(entity.getEndDateTime()),
            entity.getInstanceMinCount(),
            entity.getInstanceMaxCount(),
            entity.getDefaultInstanceMinCount(),
            entity.getDefaultInstanceMaxCount(),
            entity.getInitialMinInstanceCount(),
            entity.getGuid()
          };

      jdbcTemplate.update(
          "INSERT INTO app_scaling_specific_date_schedule (app_id, timezone, start_date_time,"
              + " end_date_time, instance_min_count, instance_max_count,"
              + " default_instance_min_count, default_instance_max_count,"
              + " initial_min_instance_count, guid) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
          objects);
    }
  }

  public void insertRecurringSchedule(List<RecurringScheduleEntity> entities) {
    JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);

    for (RecurringScheduleEntity entity : entities) {

      Object[] objects =
          new Object[] {
            entity.getAppId(),
            entity.getTimeZone(),
            entity.getDefaultInstanceMinCount(),
            entity.getDefaultInstanceMaxCount(),
            entity.getInstanceMinCount(),
            entity.getInstanceMaxCount(),
            entity.getInitialMinInstanceCount(),
            entity.getStartDate(),
            entity.getEndDate(),
            Time.valueOf(entity.getStartTime()),
            Time.valueOf(entity.getEndTime()),
            convertArrayToBits(entity.getDaysOfWeek()),
            convertArrayToBits(entity.getDaysOfMonth()),
            entity.getGuid()
          };

      jdbcTemplate.update(
          "INSERT INTO app_scaling_recurring_schedule ( app_id, timezone,"
              + " default_instance_min_count, default_instance_max_count, instance_min_count,"
              + " instance_max_count, initial_min_instance_count, start_date, end_date, start_time,"
              + " end_time, days_of_week, days_of_month, guid) VALUES ( ?, ?, ?, ?, ?, ?, ?, ?, ?,"
              + " ?, ?, ?, ?, ?)",
          objects);
    }
  }

  public void insertActiveSchedule(
      String appId,
      Long scheduleId,
      int instanceMinCount,
      int instanceMaxCount,
      int initialMinInstanceCount,
      Long startJobIdentifier) {
    JdbcTemplate jdbcTemplate = new JdbcTemplate(dataSource);

    Object[] objects =
        new Object[] {
          scheduleId,
          appId,
          startJobIdentifier,
          instanceMinCount,
          instanceMaxCount,
          initialMinInstanceCount
        };

    jdbcTemplate.update(
        "INSERT INTO app_scaling_active_schedule (id, app_id, start_job_identifier,"
            + " instance_min_count, instance_max_count, initial_min_instance_count) VALUES (?, ?,"
            + " ?, ?, ?, ?)",
        objects);
  }

  @Transactional(value = "policyDbTransactionManager")
  public void insertPolicyJson(String appId, String guid) throws Exception {
    JdbcTemplate policyDbJdbcTemplate = new JdbcTemplate(policyDbDataSource);
    Object[] objects = new Object[] {appId, PolicyUtil.getPolicyJsonContent(), guid};
    String sqlPostgresql =
        "INSERT INTO policy_json(app_id, policy_json, guid) VALUES (?, to_json(?::json), ?)";
    String sqlMysql = "INSERT INTO policy_json(app_id, policy_json, guid) VALUES (?, ?, ?)";
    String sql = null;
    if (this.getDatabaseTypeFromDataSource() == DatabaseType.POSTGRESQL) {
      sql = sqlPostgresql;
    } else if (this.getDatabaseTypeFromDataSource() == DatabaseType.MYSQL) {
      sql = sqlMysql;
    }
    policyDbJdbcTemplate.update(sql, objects);
  }

  @Transactional(value = "policyDbTransactionManager")
  public void removeAllPolicyJson() {
    JdbcTemplate policyDbJdbcTemplate = new JdbcTemplate(policyDbDataSource);
    policyDbJdbcTemplate.update("DELETE FROM policy_json");
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

  public enum DatabaseType {
    POSTGRESQL,
    MYSQL,
  }

  public DatabaseType getDatabaseTypeFromDataSource() throws Exception {
    if (this.databaseType != null) {
      return this.databaseType;
    }
    String driverName = this.dataSource.getConnection().getMetaData().getDriverName().toLowerCase();
    if (driverName != null && !driverName.isEmpty()) {
      if (driverName.contains("postgresql")) {
        this.databaseType = DatabaseType.POSTGRESQL;
        return this.databaseType;
      } else if (driverName.contains("mysql")) {
        this.databaseType = DatabaseType.MYSQL;
        return this.databaseType;
      } else {
        throw new Exception("can not support the database driver:" + driverName);
      }
    } else {
      throw new Exception("can not get database driver from datasource");
    }
  }
}
