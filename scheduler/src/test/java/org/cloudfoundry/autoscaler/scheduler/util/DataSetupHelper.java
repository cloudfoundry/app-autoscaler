package org.cloudfoundry.autoscaler.scheduler.util;

import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Date;
import java.util.List;

import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationScalingSchedules;
import org.cloudfoundry.autoscaler.scheduler.rest.model.SpecificDateSchedule;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;

/**
 * @author Fujitsu
 *
 */
public class DataSetupHelper {

	private static String startDate[] = { "2100-07-20", "2100-07-22", "2100-07-25", "2001-01-01", "2100-8-10" };
	private static String endDate[] = { "2100-07-20", "2100-07-23", "2100-07-27", "2001-01-01", "2100-8-10" };
	private static String startTime[] = { "08:00:00", "13:00:00", "09:00:00" };
	private static String endTime[] = { "10:00:00", "09:00:00", "09:00:00" };
	private static String timeZone = "(GMT+09: 00)Asia/Shanghai";
	private static String appId = "1";

	public static ScheduleEntity generateScheduleEntity() {
		ScheduleEntity entity = convertScheduleModelToScheduleEntity(generateSchedules(1)).get(0);

		return entity;
	}

	public static ApplicationScalingSchedules generateSchedules(int noOfSchedules) {
		ApplicationScalingSchedules schedules = new ApplicationScalingSchedules();
		schedules.setApp_id(appId);
		schedules.setTimezone(timeZone);

		schedules.setSpecific_date(generateSpecificDateSchedules(noOfSchedules));
		return schedules;

	}

	public static String generateJsonSchedule(int noOfSimpleSchedulesToSetUp, int noOfCronSchedulesToSetUp)
			throws JsonProcessingException {
		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = new ApplicationScalingSchedules();
		schedules.setApp_id(appId);
		schedules.setTimezone(timeZone);
		schedules.setSpecific_date(generateSpecificDateSchedules(noOfSimpleSchedulesToSetUp));

		return mapper.writeValueAsString(schedules);
	}

	private static List<ScheduleEntity> convertScheduleModelToScheduleEntity(ApplicationScalingSchedules schedules) {
		if (schedules.getSpecific_date() == null || schedules.getSpecific_date().isEmpty()) {
			return null;
		}

		List<ScheduleEntity> entities = new ArrayList<>();

		for (SpecificDateSchedule schedule : schedules.getSpecific_date()) {
			ScheduleEntity entity = new ScheduleEntity();
			entity.setAppId(schedules.getApp_id());
			entity.setTimezone(schedules.getTimezone());

			entity.setStartDate(java.sql.Date.valueOf(schedule.getStart_date()));
			entity.setEndDate(java.sql.Date.valueOf(schedule.getEnd_date()));

			entity.setStartTime(java.sql.Time.valueOf(schedule.getStart_time()));
			entity.setEndTime(java.sql.Time.valueOf(schedule.getEnd_time()));

			entity.setInstanceMinCount(schedule.getInstance_min_count());
			entity.setInstanceMaxCount(schedule.getInstance_max_count());

			entities.add(entity);
		}
		return entities;
	}

	private static List<SpecificDateSchedule> generateSpecificDateSchedules(int specificDateNum) {
		if (specificDateNum <= 0) {
			return null;
		}
		List<SpecificDateSchedule> specificDateSchedules = new ArrayList<>();

		int pos = 0;
		for (int i = 0; i < specificDateNum; i++) {
			SpecificDateSchedule schedule = new SpecificDateSchedule();

			schedule.setStart_date(getDate(startDate, pos));
			schedule.setEnd_date(getDate(endDate, pos));
			schedule.setStart_time(getTime(startTime, pos));
			schedule.setEnd_time(getTime(endTime, pos));

			schedule.setInstance_min_count(i);
			schedule.setInstance_max_count(i + 1);

			specificDateSchedules.add(schedule);
			pos++;
		}

		return specificDateSchedules;
	}

	private static String getDate(String[] date, int pos) {
		if (date != null && date.length > pos) {
			return date[pos];
		} else {
			return getCurrentDate();
		}
	}

	private static String getTime(String[] time, int pos) {
		if (time != null && time.length > pos) {
			return time[pos];
		} else {
			return getCurrentTime();
		}
	}

	private static String getCurrentTime() {
		SimpleDateFormat sdfDate = new SimpleDateFormat("HH:mm:ss");
		Date now = new Date();
		String strDate = sdfDate.format(now);
		return strDate;
	}

	private static String getCurrentDate() {
		SimpleDateFormat sdfDate = new SimpleDateFormat("yyyy-MM-dd");
		Date now = new Date();
		String strDate = sdfDate.format(now);
		return strDate;
	}

	public static String[] getStartDate() {
		return startDate;
	}

	public static String[] getEndDate() {
		return endDate;
	}

	public static String[] getStartTime() {
		return startTime;
	}

	public static String[] getEndTime() {
		return endTime;
	}

	public static String getTimeZone() {
		return timeZone;
	}

	public static String getAppId() {
		return appId;
	}

}
