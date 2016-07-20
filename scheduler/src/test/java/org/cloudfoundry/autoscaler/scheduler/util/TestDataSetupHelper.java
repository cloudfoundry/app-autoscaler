package org.cloudfoundry.autoscaler.scheduler.util;

import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Date;
import java.util.List;
import java.util.UUID;
import java.util.concurrent.TimeUnit;

import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationScalingSchedules;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;

/**
 * 
 *
 */
public class TestDataSetupHelper {
	private static String[] appIds = { "89d155a3-1879-4ba3-843e-ddad7b5c92c8", "1e05ff52-9acf-4a9f-96e4-99fe8805d494",
			"2332bd64-ab18-47f5-9813-16eb0fda5930", "cc1c9a0a-50a0-4912-92df-bea753fd93ba",
			"fa1c9a0a-50a0-4912-9211-bea753fd944a", "bc1c9a0a-50a0-4912-92df-baa753fd93ba" };
	private static List<String> genAppIds = new ArrayList<>();
	private static String timeZone = DateHelper.supportedTimezones[0];
	private static String invalidTimezone = "Invalid TimeZone";
	private static String startDate[] = { "2100-07-20", "2100-07-22", "2100-07-25", "2100-07-28", "2100-8-10" };
	private static String endDate[] = { "2100-07-20", "2100-07-23", "2100-07-27", "2100-08-07", "2100-8-10" };
	private static String startTime[] = { "08:00:00", "13:00:00", "09:00:00" };
	private static String endTime[] = { "10:00:00", "09:00:00", "09:00:00" };

	public static List<ScheduleEntity> generateSpecificDateScheduleEntities(String appId,
			int noOfSpecificDateSchedulesToSetUp) {
		List<ScheduleEntity> specificDateSchedules = generateSpecificDateScheduleEntities(appId, timeZone,
				noOfSpecificDateSchedulesToSetUp, false, 1, 5);

		return specificDateSchedules;
	}

	public static List<ScheduleEntity> generateSpecificDateScheduleEntitiesWithCurrentStartEndTime(String appId,
			int noOfSpecificDateSchedulesToSetUp) {
		List<ScheduleEntity> specificDateSchedules = generateSpecificDateScheduleEntities(appId, timeZone,
				noOfSpecificDateSchedulesToSetUp, true, 1, 5);

		return specificDateSchedules;
	}

	public static ApplicationScalingSchedules generateSpecificDateSchedules(String appId, int noOfSchedules) {
		ApplicationScalingSchedules schedules = new ApplicationScalingSchedules();
		List<ScheduleEntity> specificDateSchedules = generateSpecificDateScheduleEntities(appId, timeZone,
				noOfSchedules, false, 1, 5);
		schedules.setSpecific_date(specificDateSchedules);
		return schedules;

	}

	public static ApplicationScalingSchedules generateSpecificDateSchedulesForScheduleController(String appId,
			int noOfSpecificDateSchedules) {
		ApplicationScalingSchedules schedules = new ApplicationScalingSchedules();
		schedules.setTimeZone(timeZone);
		schedules.setInstance_min_count(1);
		schedules.setInstance_max_count(5);
		List<ScheduleEntity> specificDateSchedules = generateSpecificDateScheduleEntities(appId, null,
				noOfSpecificDateSchedules, false, null, null);
		schedules.setSpecific_date(specificDateSchedules);
		return schedules;

	}

	public static String generateJsonSchedule(String appId, int noOfSpecificDateSchedulesToSetUp,
			int noOfRecurringSchedulesToSetUp) throws JsonProcessingException {
		ObjectMapper mapper = new ObjectMapper();

		ApplicationScalingSchedules schedules = new ApplicationScalingSchedules();
		schedules.setTimeZone(timeZone);
		schedules.setInstance_min_count(1);
		schedules.setInstance_max_count(5);
		schedules.setSpecific_date(
				generateSpecificDateScheduleEntities(null, null, noOfSpecificDateSchedulesToSetUp, false, null, null));

		return mapper.writeValueAsString(schedules);
	}

	private static List<ScheduleEntity> generateSpecificDateScheduleEntities(String appId, String timeZone,
			int noOfSpecificDateSchedulesToSetUp, boolean setCurrentDateTime, Integer defaultInstanceMinCount,
			Integer defaultInstanceMaxCount) {
		if (noOfSpecificDateSchedulesToSetUp <= 0) {
			return null;
		}
		List<ScheduleEntity> specificDateSchedules = new ArrayList<>();

		int pos = 0;
		for (int i = 0; i < noOfSpecificDateSchedulesToSetUp; i++) {
			ScheduleEntity specificDateScheduleEntity = new ScheduleEntity();
			specificDateScheduleEntity.setAppId(appId);
			specificDateScheduleEntity.setTimeZone(timeZone);

			if (setCurrentDateTime) {
				specificDateScheduleEntity.setStartDate(java.sql.Date.valueOf(getCurrentDate(0)));
				specificDateScheduleEntity.setEndDate(java.sql.Date.valueOf(getCurrentDate(0)));
				specificDateScheduleEntity.setStartTime(java.sql.Time.valueOf(getCurrentTime(0)));
				specificDateScheduleEntity.setEndTime(java.sql.Time.valueOf(getCurrentTime(0)));
			} else {
				specificDateScheduleEntity.setStartDate(java.sql.Date.valueOf(getDate(startDate, pos, 0)));
				specificDateScheduleEntity.setEndDate(java.sql.Date.valueOf(getDate(endDate, pos, 5)));
				specificDateScheduleEntity.setStartTime(java.sql.Time.valueOf(getTime(startTime, pos, 0)));
				specificDateScheduleEntity.setEndTime(java.sql.Time.valueOf(getTime(endTime, pos, 5)));
			}

			specificDateScheduleEntity.setInstanceMinCount(i + 5);
			specificDateScheduleEntity.setInstanceMaxCount(i + 6);
			specificDateScheduleEntity.setDefaultInstanceMinCount(defaultInstanceMinCount);
			specificDateScheduleEntity.setDefaultInstanceMaxCount(defaultInstanceMaxCount);
			specificDateScheduleEntity.setScheduleType(ScheduleTypeEnum.SPECIFIC_DATE.getDbValue());
			specificDateSchedules.add(specificDateScheduleEntity);
			pos++;
		}

		return specificDateSchedules;
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
