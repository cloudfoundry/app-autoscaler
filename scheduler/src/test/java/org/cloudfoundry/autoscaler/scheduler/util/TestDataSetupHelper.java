package org.cloudfoundry.autoscaler.scheduler.util;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.time.LocalDate;
import java.time.LocalDateTime;
import java.time.LocalTime;
import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.time.format.DateTimeFormatter;
import java.time.temporal.ChronoUnit;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Calendar;
import java.util.Date;
import java.util.List;
import java.util.Random;
import java.util.TimeZone;
import java.util.UUID;
import java.util.concurrent.TimeUnit;
import org.cloudfoundry.autoscaler.scheduler.entity.ActiveScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.quartz.AppScalingScheduleJob;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationSchedules;
import org.cloudfoundry.autoscaler.scheduler.rest.model.Schedules;
import org.quartz.JobBuilder;
import org.quartz.JobDataMap;
import org.quartz.JobDetail;
import org.quartz.JobKey;
import org.quartz.Trigger;
import org.quartz.TriggerKey;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/** Class to set up the test data for the test classes */
public class TestDataSetupHelper {
  private static Class<?> clazz = TestDataSetupHelper.class;
  private Logger logger = LoggerFactory.getLogger(this.getClass());

  private static List<String> genAppIds = new ArrayList<>();
  public static String timeZone = "GMT";

  private static String currentStartDateTime =
      getCurrentDateOrTime(5, DateHelper.DATE_TIME_FORMAT, getTimeZone());
  private static String currentEndDateTime =
      getCurrentDateOrTime(6, DateHelper.DATE_TIME_FORMAT, getTimeZone());

  private static String[] startDateTime = {
    currentStartDateTime,
    "2100-07-22T13:00",
    "2100-07-25T09:00",
    "2100-07-28T00:00",
    "2100-8-10T00:00"
  };
  private static String[] endDateTime = {
    currentEndDateTime,
    "2100-07-23T09:00",
    "2100-07-27T09:00",
    "2100-08-07T00:00",
    "2100-8-11T00:00"
  };

  private static String[] startTime = {"00:00", "02:00", "10:00", "11:00", "23:00"};
  private static String[] endTime = {"01:00", "08:00", "10:01", "12:00", "23:59"};

  public static ApplicationSchedules generateApplicationPolicy(
      int noOfSpecificDateSchedules, int noOfRecurringSchedules) {

    int noOfDomRecurringSchedules = noOfRecurringSchedules / 2;
    int noOfDowRecurringSchedules = noOfRecurringSchedules - noOfDomRecurringSchedules;

    return new ApplicationPolicyBuilder(
            1,
            5,
            timeZone,
            noOfSpecificDateSchedules,
            noOfDomRecurringSchedules,
            noOfDowRecurringSchedules)
        .build();
  }

  public static Schedules generateSchedulesWithEntitiesOnly(
      String appId,
      String guid,
      boolean generateScheduleId,
      int noOfSpecificSchedules,
      int noOfDomRecurringSchedules,
      int noOfDowRecurringSchedules) {

    List<SpecificDateScheduleEntity> specificDateSchedules =
        generateSpecificDateScheduleEntities(
            appId, guid, generateScheduleId, noOfSpecificSchedules);

    List<RecurringScheduleEntity> recurringSchedules =
        generateRecurringScheduleEntities(
            appId, guid, generateScheduleId, noOfDomRecurringSchedules, noOfDowRecurringSchedules);

    return new ScheduleBuilder()
        .setSpecificDate(specificDateSchedules)
        .setRecurringSchedule(recurringSchedules)
        .build();
  }

  public static List<SpecificDateScheduleEntity> generateSpecificDateScheduleEntities(
      String appId, String guid, boolean generateScheduleId, int noOfSpecificDateSchedulesToSetUp) {
    SpecificDateScheduleEntitiesBuilder builder =
        new SpecificDateScheduleEntitiesBuilder(noOfSpecificDateSchedulesToSetUp);
    if (generateScheduleId) {
      builder.setScheduleId();
    }
    return builder
        .setAppid(appId)
        .setGuid(guid)
        .setTimeZone(timeZone)
        .setDefaultInstanceMinCount(1)
        .setDefaultInstanceMaxCount(5)
        .build();
  }

  public static List<RecurringScheduleEntity> generateRecurringScheduleEntities(
      String appId,
      String guid,
      boolean generateScheduleId,
      int noOfDomRecurringSchedules,
      int noOfDowRecurringSchedules) {

    RecurringScheduleEntitiesBuilder builder =
        new RecurringScheduleEntitiesBuilder(noOfDomRecurringSchedules, noOfDowRecurringSchedules);
    if (generateScheduleId) {
      builder.setScheduleId();
    }
    return builder
        .setAppId(appId)
        .setGuid(guid)
        .setTimeZone(timeZone)
        .setDefaultInstanceMinCount(1)
        .setDefaultInstanceMaxCount(5)
        .build();
  }

  public static String generateJsonSchedule(
      int noOfSpecificDateSchedulesToSetUp, int noOfRecurringSchedulesToSetUp)
      throws JsonProcessingException {
    ObjectMapper mapper = new ObjectMapper();

    ApplicationSchedules applicationPolicy =
        generateApplicationPolicy(noOfSpecificDateSchedulesToSetUp, noOfRecurringSchedulesToSetUp);

    return mapper.writeValueAsString(applicationPolicy);
  }

  public static String generateJsonForOverlappingRecurringScheduleWithStartEndDate(
      String firstStartDateStr,
      String firstEndDateStr,
      String secondStartDateStr,
      String secondEndDateStr)
      throws Exception {
    ObjectMapper mapper = new ObjectMapper();

    int[] dayOfWeek = {1, 2, 3, 4, 5, 6, 7};
    LocalTime firstStartTime = LocalTime.parse("00:00");
    LocalTime firstEndTime = LocalTime.parse("22:00:00");
    LocalTime secondStartTime = firstEndTime;
    LocalTime secondEndTime = LocalTime.parse("23:59:00");

    List<RecurringScheduleEntity> entities =
        new RecurringScheduleEntitiesBuilder(0, 2)
            // Set data in first entity
            .setDayOfWeek(0, dayOfWeek)
            .setDayOfMonth(0, null)
            .setStartDate(0, getDate(firstStartDateStr))
            .setEndDate(0, getDate(firstEndDateStr))
            .setStartTime(0, firstStartTime)
            .setEndTime(0, firstEndTime)
            // Set data in second entity
            .setDayOfWeek(1, dayOfWeek)
            .setDayOfMonth(1, null)
            .setStartDate(1, getDate(secondStartDateStr))
            .setEndDate(1, getDate(secondEndDateStr))
            .setStartTime(1, secondStartTime)
            .setEndTime(1, secondEndTime)
            .build();

    Schedules schedules =
        new ScheduleBuilder(timeZone, 0, 0, 0).setRecurringSchedule(entities).build();
    ApplicationSchedules applicationPolicy = generateApplicationPolicy(0, 1);
    applicationPolicy.setSchedules(schedules);

    return mapper.writeValueAsString(applicationPolicy);
  }

  public static ActiveScheduleEntity generateActiveScheduleEntity(String appId, Long scheduleId) {

    ActiveScheduleEntity activeScheduleEntity = new ActiveScheduleEntity();

    activeScheduleEntity.setAppId(appId);
    activeScheduleEntity.setId(scheduleId);
    activeScheduleEntity.setInstanceMinCount(2);
    activeScheduleEntity.setInstanceMaxCount(4);
    activeScheduleEntity.setInitialMinInstanceCount(null);
    activeScheduleEntity.setStartJobIdentifier(new Date().getTime());
    return activeScheduleEntity;
  }

  private static LocalDate getDate(String dateStr) throws ParseException {
    DateTimeFormatter dateTimeFormatter = DateTimeFormatter.ofPattern(DateHelper.DATE_FORMAT);

    if (dateStr != null) {
      return LocalDate.parse(dateStr, dateTimeFormatter);
    }
    return null;
  }

  static String getDateString(String[] date, int pos, int offsetMin) {
    if (date != null && date.length > pos) {
      return date[pos];
    } else {
      return getCurrentDateOrTime(offsetMin, DateHelper.DATE_FORMAT, getTimeZone());
    }
  }

  public static String convertDateTimeString(Date dateTime, TimeZone timeZone) {
    SimpleDateFormat sdf = new SimpleDateFormat(DateHelper.DATE_TIME_FORMAT);
    sdf.setTimeZone(timeZone);
    return sdf.format(dateTime);
  }

  public static LocalTime getZoneTimeWithOffset(int offsetMin) {
    return LocalDateTime.now(ZoneId.of(TestDataSetupHelper.timeZone))
        .plusMinutes(offsetMin)
        .toLocalTime()
        .truncatedTo(ChronoUnit.MINUTES);
  }

  private static String getCurrentDateOrTime(int offsetMin, String format, String timeZone) {
    SimpleDateFormat sdfDate = new SimpleDateFormat(format);
    Calendar calNow = Calendar.getInstance();
    calNow.add(Calendar.MINUTE, offsetMin);

    sdfDate.setTimeZone(TimeZone.getTimeZone(timeZone));
    return sdfDate.format(calNow.getTime());
  }

  public static LocalDate getZoneDateNow() {
    TimeZone timeZone = TimeZone.getTimeZone(TestDataSetupHelper.timeZone);
    return ZonedDateTime.now(timeZone.toZoneId()).toLocalDate();
  }

  public static Date getCurrentDateTime(int offsetMin) {
    return new Date(System.currentTimeMillis() + TimeUnit.MINUTES.toMillis(offsetMin));
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

  public static String generateGuid() {
    return UUID.randomUUID().toString();
  }

  public static int[] generateDayOfWeek() {
    int arraySize = (int) (new Date().getTime() % 7) + 1;
    int today = LocalDateTime.now(ZoneId.of(timeZone)).getDayOfWeek().getValue();
    int[] array =
        makeRandomArray(
            new Random(Calendar.getInstance().getTimeInMillis()),
            arraySize,
            DateHelper.DAY_OF_WEEK_MINIMUM,
            DateHelper.DAY_OF_WEEK_MAXIMUM,
            today);
    return array;
  }

  public static int[] generateDayOfMonth() {
    int arraySize = (int) (new Date().getTime() % 31) + 1;
    int today = LocalDateTime.now(ZoneId.of(timeZone)).getDayOfMonth();
    int[] array =
        makeRandomArray(
            new Random(Calendar.getInstance().getTimeInMillis()),
            arraySize,
            DateHelper.DAY_OF_MONTH_MINIMUM,
            DateHelper.DAY_OF_MONTH_MAXIMUM,
            today);
    return array;
  }

  private static int[] makeRandomArray(
      Random rand, int size, int randMin, int randMax, int fixValue) {
    int[] array = rand.ints(randMin, randMax + 1).distinct().limit(size).toArray();
    Arrays.sort(array);
    if (Arrays.binarySearch(array, fixValue) < 0) {
      array[0] = fixValue;
    }
    Arrays.sort(array);
    return array;
  }

  public static int convertIntToCalendarDayOfWeek(int dayOfWeek) {
    return dayOfWeek == Calendar.SUNDAY ? 7 : dayOfWeek - 1;
  }

  public static JobDataMap setupJobDataMap(JobDetail jobDetail) {
    String appId = TestDataSetupHelper.generateAppIds(1)[0];
    Long scheduleId = 1L;

    JobDataMap jobDataMap = jobDetail.getJobDataMap();
    jobDataMap.put(ScheduleJobHelper.APP_ID, appId);
    jobDataMap.put(ScheduleJobHelper.SCHEDULE_ID, scheduleId);
    jobDataMap.put(ScheduleJobHelper.TIMEZONE, TimeZone.getDefault().getID());
    jobDataMap.put(ScheduleJobHelper.INITIAL_MIN_INSTANCE_COUNT, 1);
    jobDataMap.put(ScheduleJobHelper.INSTANCE_MIN_COUNT, 2);
    jobDataMap.put(ScheduleJobHelper.INSTANCE_MAX_COUNT, 4);
    jobDataMap.put(ScheduleJobHelper.DEFAULT_INSTANCE_MIN_COUNT, 1);
    jobDataMap.put(ScheduleJobHelper.DEFAULT_INSTANCE_MAX_COUNT, 5);

    jobDataMap.put(ScheduleJobHelper.RescheduleCount.ACTIVE_SCHEDULE.name(), 1);
    jobDataMap.put(ScheduleJobHelper.RescheduleCount.SCALING_ENGINE_NOTIFICATION.name(), 1);
    jobDataMap.put(ScheduleJobHelper.ACTIVE_SCHEDULE_TABLE_CREATE_TASK_DONE, false);
    jobDataMap.put(ScheduleJobHelper.CREATE_END_JOB_TASK_DONE, false);

    return jobDataMap;
  }

  public static List<String> getAllGeneratedAppIds() {
    return genAppIds;
  }

  public static String getSchedulerPath(String appId) {
    return String.format("/v1/apps/%s/schedules", appId);
  }

  static String getTimeZone() {
    return timeZone;
  }

  static String[] getStartDateTime() {
    return startDateTime;
  }

  static String[] getEndDateTime() {
    return endDateTime;
  }

  static String[] getStarTime() {
    return startTime;
  }

  static String[] getEndTime() {
    return endTime;
  }

  public static class JobInformation<T extends AppScalingScheduleJob> {
    private JobDetail jobDetail;
    private Trigger trigger;

    public JobInformation(Class<T> appScalingScheduleJobClass) {
      JobKey jobKey =
          new JobKey("TestJobKey", ScheduleTypeEnum.SPECIFIC_DATE.getScheduleIdentifier());
      this.jobDetail =
          JobBuilder.newJob(appScalingScheduleJobClass).withIdentity(jobKey).storeDurably().build();

      TriggerKey triggerKey =
          new TriggerKey("TestTriggerKey", ScheduleTypeEnum.SPECIFIC_DATE.getScheduleIdentifier());

      this.trigger = ScheduleJobHelper.buildTrigger(triggerKey, jobKey, ZonedDateTime.now());
    }

    public JobDetail getJobDetail() {
      return jobDetail;
    }

    public Trigger getTrigger() {
      return trigger;
    }
  }
}
