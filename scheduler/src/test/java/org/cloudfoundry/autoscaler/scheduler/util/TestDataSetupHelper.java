package org.cloudfoundry.autoscaler.scheduler.util;

import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Calendar;
import java.util.Date;
import java.util.List;
import java.util.Random;
import java.util.UUID;
import java.util.concurrent.TimeUnit;

import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.Logger;
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
	private static Class<?> clazz = TestDataSetupHelper.class;
	private static Logger logger = LogManager.getLogger(clazz);

	private static List<String> genAppIds = new ArrayList<>();
	private static String timeZone = DateHelper.supportedTimezones[81];
	private static String invalidTimezone = "Invalid TimeZone";

	private static String startDateTime[] = { "2100-07-20T08:00", "2100-07-22T13:00", "2100-07-25T09:00",
			"2100-07-28T00:00", "2100-8-10T00:00" };
	private static String endDateTime[] = { "2100-07-20T10:00", "2100-07-23T09:00", "2100-07-27T09:00",
			"2100-08-07T00:00", "2100-8-11T00:00" };

	private static String startTime[] = { "00:00:00", "2:00:00", "10:00:00", "11:00:12", "23:00:00" };
	private static String endTime[] = { "1:00:00", "8:00:00", "10:01:00", "12:00:00", "23:59:00" };

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
		return generateSpecificDateScheduleEntities(appId, timeZone, noOfSpecificDateSchedulesToSetUp,
				isStartEndDateTimeCurrentDateTime, 1, 5);
	}

	private static List<SpecificDateScheduleEntity> generateSpecificDateScheduleEntities(String appId, String timeZone,
			int noOfSpecificDateSchedulesToSetUp, boolean isStartEndDateTimeCurrentDateTime,
			Integer defaultInstanceMinCount, Integer defaultInstanceMaxCount) {
		if (noOfSpecificDateSchedulesToSetUp <= 0) {
			return null;
		}
		List<SpecificDateScheduleEntity> specificDateSchedules = new ArrayList<>();

		int pos = 0;
		SimpleDateFormat sdf = new SimpleDateFormat(DateHelper.DATE_TIME_FORMAT);
		for (int i = 0; i < noOfSpecificDateSchedulesToSetUp; i++) {
			SpecificDateScheduleEntity specificDateScheduleEntity = new SpecificDateScheduleEntity();
			specificDateScheduleEntity.setAppId(appId);
			specificDateScheduleEntity.setTimeZone(timeZone);

			try {
				if (isStartEndDateTimeCurrentDateTime) {
					specificDateScheduleEntity.setStartDateTime(sdf.parse(getCurrentDateTime(0)));
					specificDateScheduleEntity.setEndDateTime(sdf.parse(getCurrentDateTime(0)));
				} else {
					specificDateScheduleEntity.setStartDateTime(sdf.parse(getDateString(startDateTime, pos, 0)));
					specificDateScheduleEntity.setEndDateTime(sdf.parse(getDateString(endDateTime, pos, 5)));
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
		return generateRecurringScheduleEntities(appId, timeZone, noOfRecurringSchedulesToSetUp, false, 1, 5,
				isDayOfWeek);
	}

	private static List<RecurringScheduleEntity> generateRecurringScheduleEntities(String appId, String timeZone,
			int noOfRecurringSchedulesToSetUp, boolean isStartEndDateTimeCurrentDateTime,
			Integer defaultInstanceMinCount, Integer defaultInstanceMaxCount, boolean isDayOfWeek) {
		List<RecurringScheduleEntity> recurringSchedules = new ArrayList<>();

		int pos = 0;
		for (int i = 0; i < noOfRecurringSchedulesToSetUp; i++) {
			RecurringScheduleEntity recurringScheduleEntity = new RecurringScheduleEntity();
			recurringScheduleEntity.setAppId(appId);
			recurringScheduleEntity.setTimeZone(timeZone);
			if (isStartEndDateTimeCurrentDateTime) {
				recurringScheduleEntity.setStartTime(java.sql.Time.valueOf(getCurrentTime(0)));
				recurringScheduleEntity.setEndTime(java.sql.Time.valueOf(getCurrentTime(0)));
			} else {
				recurringScheduleEntity.setStartTime(java.sql.Time.valueOf(getTimeString(startTime, pos, 0)));
				recurringScheduleEntity.setEndTime(java.sql.Time.valueOf(getTimeString(endTime, pos, 5)));
			}

			if (isDayOfWeek) {
				recurringScheduleEntity.setDayOfWeek(generateDayOfWeek());
			} else {
				recurringScheduleEntity.setDayOfMonth(generateDayOfMonth());
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

		int noOfDayOfWeek = noOfRecurringSchedulesToSetUp % 2;
		int noOfDayOfMonth = noOfRecurringSchedulesToSetUp - noOfDayOfWeek;
		List<RecurringScheduleEntity> recurringScheduleEntities = generateRecurringScheduleEntities(null, null,
				noOfDayOfWeek, false, null, null, true);
		recurringScheduleEntities
				.addAll(generateRecurringScheduleEntities(null, null, noOfDayOfMonth, false, null, null, false));

		schedules.setRecurring_schedule(recurringScheduleEntities);
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

	public static String generateJsonScheduleWithStartEndDate(String firstStartDateStr, String firstEndDateStr,
			String secondStartDateStr, String secondEndDateStr) throws Exception {
		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 2;
		ApplicationScalingSchedules schedules = TestDataSetupHelper.generateSchedulesForRestApi(0,
				noOfRecurringSchedulesToSetUp);

		// Overlap recurring schedules.
		RecurringScheduleEntity firstEntity = schedules.getRecurring_schedule().get(0);
		RecurringScheduleEntity secondEntity = schedules.getRecurring_schedule().get(1);
		secondEntity.setStartTime(firstEntity.getEndTime());

		firstEntity.setDayOfWeek(new int[] { 1, 2, 3, 4, 5, 6, 7 });
		firstEntity.setDayOfMonth(null);

		secondEntity.setDayOfWeek(firstEntity.getDayOfWeek());
		secondEntity.setDayOfMonth(null);

		firstEntity.setStartDate(getDate(firstStartDateStr));
		firstEntity.setEndDate(getDate(firstEndDateStr));

		secondEntity.setStartDate(getDate(secondStartDateStr));
		secondEntity.setEndDate(getDate(secondEndDateStr));

		return mapper.writeValueAsString(schedules);
	}

	private static Date getDate(String dateStr) throws Exception {
		SimpleDateFormat sdf = new SimpleDateFormat(DateHelper.DATE_FORMAT);
		Date date = null;
		if (dateStr != null) {
			date = sdf.parse(dateStr);
		}
		return date;
	}

	private static String getDateString(String[] date, int pos, int offsetMin) {
		if (date != null && date.length > pos) {
			return date[pos];
		} else {
			return getCurrentDate(offsetMin);
		}
	}

	private static String getTimeString(String[] time, int pos, int offsetMin) {
		if (time != null && time.length > pos) {
			return time[pos];
		} else {
			return getCurrentTime(offsetMin);
		}
	}

	private static String getCurrentDateTime(int offsetMin) {
		SimpleDateFormat sdfDate = new SimpleDateFormat(DateHelper.DATE_TIME_FORMAT);
		Date now = new Date();
		now.setTime(now.getTime() + TimeUnit.MINUTES.toMillis(offsetMin));
		return sdfDate.format(now);
	}

	private static String getCurrentTime(int offsetMin) {
		SimpleDateFormat sdfDate = new SimpleDateFormat(DateHelper.TIME_FORMAT);
		Calendar calNow = Calendar.getInstance();
		calNow.add(Calendar.MINUTE, offsetMin);
		return sdfDate.format(calNow.getTime());
	}

	private static String getCurrentDate(int offsetMin) {
		SimpleDateFormat sdfDate = new SimpleDateFormat(DateHelper.DATE_FORMAT);
		Calendar calNow = Calendar.getInstance();
		calNow.add(Calendar.MINUTE, offsetMin);
		return sdfDate.format(calNow.getTime());
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

	public static int[] generateDayOfWeek() {
		int arraySize = (int) (new Date().getTime() % 7) + 1;
		int[] array = makeRandomArray(new Random(Calendar.getInstance().getTimeInMillis()), arraySize,
				DateHelper.DAY_OF_WEEK_MINIMUM, DateHelper.DAY_OF_WEEK_MAXMUM);
		logger.debug("Generate day of week array:" + Arrays.toString(array));
		return array;
	}

	public static int[] generateDayOfMonth() {
		int arraySize = (int) (new Date().getTime() % 31) + 1;
		int[] array = makeRandomArray(new Random(Calendar.getInstance().getTimeInMillis()), arraySize,
				DateHelper.DAY_OF_MONTH_MINIMUM, DateHelper.DAY_OF_MONTH_MAXMUM);
		logger.debug("Generate day of month array:" + Arrays.toString(array));
		return array;
	}

	private static int[] makeRandomArray(Random rand, int size, int randMin, int randMax) {
		int[] array = rand.ints(randMin, randMax + 1).distinct().limit(size).toArray();
		Arrays.sort(array);
		return array;
	}

	public static Date addDaysToNow(int afterDays) {
		Calendar calNow = Calendar.getInstance();
		calNow.add(Calendar.DAY_OF_MONTH, afterDays);
		calNow.set(Calendar.HOUR_OF_DAY, 0);
		calNow.set(Calendar.MINUTE, 0);
		calNow.set(Calendar.SECOND, 0);
		calNow.set(Calendar.MILLISECOND, 0);
		return calNow.getTime();
	}

	public static List<String> getAllGeneratedAppIds() {
		return genAppIds;
	}

	public static String[] getStartDateTime() {
		return startDateTime;
	}

	public static String[] getEndDateTime() {
		return endDateTime;
	}

	public static String getInvalidTimezone() {
		return invalidTimezone;
	}

}
