package org.cloudfoundry.autoscaler.scheduler.util;

import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Date;
import java.util.List;
import java.util.UUID;
import java.util.concurrent.TimeUnit;

import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationScalingSchedules;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;

/**
 * Class to set up the test data for the test classes
 *
 */
public class TestDataSetupHelper {
	private static List<String> genAppIds = new ArrayList<>();
	private static String timeZone = DateHelper.supportedTimezones[0];
	private static String invalidTimezone = "Invalid TimeZone";
	private static String startDate[] = { "2100-07-20", "2100-07-22", "2100-07-25", "2100-07-28", "2100-8-10" };
	private static String endDate[] = { "2100-07-20", "2100-07-23", "2100-07-27", "2100-08-07", "2100-8-10" };
	private static String startTime[] = { "08:00:00", "13:00:00", "09:00:00" };
	private static String endTime[] = { "10:00:00", "09:00:00", "09:00:00" };

	private static int dayOfWeek[] = { 1, 2, 3, 4, 5, 6, 7 };
	private static int dayOfMonth[] = { 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 20, 30, 31 };

	public static ApplicationScalingSchedules generateSchedules(String appId, int noOfSpecificSchedules,
			int noOfRecurringScheduls) {
		ApplicationScalingSchedules schedules = new ApplicationScalingSchedules();
		List<SpecificDateScheduleEntity> specificDateSchedules = generateSpecificDateScheduleEntities(appId, timeZone,
				noOfSpecificSchedules, false, 1, 5);
		List<RecurringScheduleEntity> recurringSchedules = generateRecurringScheduleEntities(appId, timeZone,
				noOfRecurringScheduls, false, 1, 5, false);
		schedules.setSpecific_date(specificDateSchedules);
		schedules.setRecurring_schedule(recurringSchedules);
		return schedules;

	}

	public static List<SpecificDateScheduleEntity> generateSpecificDateSchedules(String appId,
			int noOfSpecificDateSchedulesToSetUp, boolean isStartEndDateTimeCurrentDateTime) {
		List<SpecificDateScheduleEntity> specificDateSchedules = generateSpecificDateScheduleEntities(appId, timeZone,
				noOfSpecificDateSchedulesToSetUp, isStartEndDateTimeCurrentDateTime, 1, 5);

		return specificDateSchedules;
	}

	private static List<SpecificDateScheduleEntity> generateSpecificDateScheduleEntities(String appId, String timeZone,
			int noOfSpecificDateSchedulesToSetUp, boolean isStartEndDateTimeCurrentDateTime,
			Integer defaultInstanceMinCount, Integer defaultInstanceMaxCount) {
		if (noOfSpecificDateSchedulesToSetUp <= 0) {
			return null;
		}
		List<SpecificDateScheduleEntity> specificDateSchedules = new ArrayList<>();

		int pos = 0;
		SimpleDateFormat sdf = new SimpleDateFormat("yyyy-MM-dd");
		for (int i = 0; i < noOfSpecificDateSchedulesToSetUp; i++) {
			SpecificDateScheduleEntity specificDateScheduleEntity = new SpecificDateScheduleEntity();
			specificDateScheduleEntity.setAppId(appId);
			specificDateScheduleEntity.setTimeZone(timeZone);

			try {
				if (isStartEndDateTimeCurrentDateTime) {
					specificDateScheduleEntity.setStartDate(sdf.parse(getCurrentDate(0)));
					specificDateScheduleEntity.setEndDate(sdf.parse(getCurrentDate(0)));
					specificDateScheduleEntity.setStartTime(java.sql.Time.valueOf(getCurrentTime(0)));
					specificDateScheduleEntity.setEndTime(java.sql.Time.valueOf(getCurrentTime(0)));
				} else {
					specificDateScheduleEntity.setStartDate(sdf.parse(getDate(startDate, pos, 0)));
					specificDateScheduleEntity.setEndDate(sdf.parse(getDate(endDate, pos, 5)));
					specificDateScheduleEntity.setStartTime(java.sql.Time.valueOf(getTime(startTime, pos, 0)));
					specificDateScheduleEntity.setEndTime(java.sql.Time.valueOf(getTime(endTime, pos, 5)));
				}
			} catch (ParseException e) {
				throw new RuntimeException(e.getMessage());
			}

			specificDateScheduleEntity.setInstanceMinCount(i + 5);
			specificDateScheduleEntity.setInstanceMaxCount(i + 6);
			specificDateScheduleEntity.setDefaultInstanceMinCount(defaultInstanceMinCount);
			specificDateScheduleEntity.setDefaultInstanceMaxCount(defaultInstanceMaxCount);
			specificDateSchedules.add(specificDateScheduleEntity);
			pos++;
		}

		return specificDateSchedules;
	}

	public static List<RecurringScheduleEntity> generateRecurringSchedules(String appId,
			int noOfRecurringSchedulesToSetUp, boolean isDayOfWeek) {
		List<RecurringScheduleEntity> recurringScheduleEntities = generateRecurringScheduleEntities(appId, timeZone,
				noOfRecurringSchedulesToSetUp, false, 1, 5, false);

		return recurringScheduleEntities;
	}

	private static List<RecurringScheduleEntity> generateRecurringScheduleEntities(String appId, String timeZone,
			int noOfRecurringSchedulesToSetUp, boolean isStartEndDateTimeCurrentDateTime,
			Integer defaultInstanceMinCount, Integer defaultInstanceMaxCount, boolean isDayOfWeek) {
		List<RecurringScheduleEntity> recurringSchedules = new ArrayList<>();

		int pos = 0;
		SimpleDateFormat sdf = new SimpleDateFormat("yyyy-MM-dd");
		for (int i = 0; i < noOfRecurringSchedulesToSetUp; i++) {
			RecurringScheduleEntity recurringScheduleEntity = new RecurringScheduleEntity();
			recurringScheduleEntity.setAppId(appId);
			recurringScheduleEntity.setTimeZone(timeZone);
			try {
				if (isStartEndDateTimeCurrentDateTime) {
					recurringScheduleEntity.setStartDate(sdf.parse(getCurrentDate(0)));
					recurringScheduleEntity.setEndDate(sdf.parse(getCurrentDate(0)));
					recurringScheduleEntity.setStartTime(java.sql.Time.valueOf(getCurrentTime(0)));
					recurringScheduleEntity.setEndTime(java.sql.Time.valueOf(getCurrentTime(0)));
				} else {
					recurringScheduleEntity.setStartDate(sdf.parse(getDate(startDate, pos, 0)));
					recurringScheduleEntity.setEndDate(sdf.parse(getDate(endDate, pos, 5)));
					recurringScheduleEntity.setStartTime(java.sql.Time.valueOf(getTime(startTime, pos, 0)));
					recurringScheduleEntity.setEndTime(java.sql.Time.valueOf(getTime(endTime, pos, 5)));
				}
			} catch (ParseException e) {
				throw new RuntimeException(e.getMessage());
			}

			if (isDayOfWeek) {
				recurringScheduleEntity.setDayOfWeek((dayOfWeek));
			} else {
				recurringScheduleEntity.setDayOfMonth((dayOfMonth));
			}
			recurringScheduleEntity.setInstanceMinCount(i + 5);
			recurringScheduleEntity.setInstanceMaxCount(i + 6);
			recurringScheduleEntity.setDefaultInstanceMinCount(defaultInstanceMinCount);
			recurringScheduleEntity.setDefaultInstanceMaxCount(defaultInstanceMaxCount);
			recurringSchedules.add(recurringScheduleEntity);
			pos++;
		}

		return recurringSchedules;
	}

	public static ApplicationScalingSchedules generateSchedulesForRestApi(int noOfSpecificDateSchedules,
			int noOfRecurringSchedulesToSetUp) {
		ApplicationScalingSchedules schedules = new ApplicationScalingSchedules();
		schedules.setTimeZone(timeZone);
		schedules.setInstance_min_count(1);
		schedules.setInstance_max_count(5);
		List<SpecificDateScheduleEntity> specificDateSchedules = generateSpecificDateScheduleEntities(null, null,
				noOfSpecificDateSchedules, false, null, null);
		schedules.setRecurring_schedule(
				generateRecurringScheduleEntities(null, null, noOfRecurringSchedulesToSetUp, false, null, null, true));
		schedules.setSpecific_date(specificDateSchedules);
		return schedules;

	}

	public static String generateJsonSchedule(String appId, int noOfSpecificDateSchedulesToSetUp,
			int noOfRecurringSchedulesToSetUp) throws JsonProcessingException {
		ObjectMapper mapper = new ObjectMapper();

		ApplicationScalingSchedules schedules = generateSchedulesForRestApi(noOfSpecificDateSchedulesToSetUp,
				noOfRecurringSchedulesToSetUp);

		return mapper.writeValueAsString(schedules);
	}

	private static String getDate(String[] date, int pos, int offsetMin) {
		if (date != null && date.length > pos) {
			return date[pos];
		} else {
			return getCurrentDate(offsetMin);
		}
	}

	private static String getTime(String[] time, int pos, int offsetMin) {
		if (time != null && time.length > pos) {
			return time[pos];
		} else {
			return getCurrentTime(offsetMin);
		}
	}

	private static String getCurrentTime(int offsetMin) {
		SimpleDateFormat sdfDate = new SimpleDateFormat("HH:mm:ss");
		Date now = new Date();
		now.setTime(now.getTime() + TimeUnit.MINUTES.toMillis(offsetMin));
		String strDate = sdfDate.format(now);
		return strDate;
	}

	private static String getCurrentDate(int offsetMin) {
		SimpleDateFormat sdfDate = new SimpleDateFormat("yyyy-MM-dd");
		Date now = new Date();
		now.setTime(now.getTime() + TimeUnit.MINUTES.toMillis(offsetMin));
		String strDate = sdfDate.format(now);
		return strDate;
	}

	public static String getTimeZone() {
		return timeZone;
	}

	public static String[] generateAppIds(int noOfAppIdsToGenerate) {
		List<String> appIds = new ArrayList<>();
		for (int i = 0; i < noOfAppIdsToGenerate; i++) {
			UUID uuid = UUID.randomUUID();
			genAppIds.add(uuid.toString());
			appIds.add(uuid.toString());
		}
		return appIds.toArray(new String[0]);
	}

	public static List<String> getAllGeneratedAppIds() {
		return genAppIds;
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

	public static String getInvalidTimezone() {
		return invalidTimezone;
	}

}
