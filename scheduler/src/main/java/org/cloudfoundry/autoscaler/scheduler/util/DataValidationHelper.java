package org.cloudfoundry.autoscaler.scheduler.util;

import java.time.LocalDate;
import java.time.LocalDateTime;
import java.time.LocalTime;
import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.HashSet;
import java.util.List;
import java.util.Set;
import java.util.TimeZone;

/** Helper class for validating the data */
public class DataValidationHelper {

  /**
   * Checks if the specified object is null.
   *
   * @param object
   * @return true if not null otherwise false
   */
  public static boolean isNotNull(Object object) {
    return object != null;
  }

  /**
   * Checks if specified string is not empty (not null and not blank)
   *
   * @param string
   * @return true or false
   */
  public static boolean isNotEmpty(String string) {
    return (isNotNull(string) && !string.isEmpty());
  }

  /**
   * Checks if specified array is not empty (not null and not empty)
   *
   * @param array
   * @return
   */
  public static boolean isNotEmpty(int[] array) {
    return (isNotNull(array) && array.length > 0);
  }

  /**
   * Checks if the timezone is valid
   *
   * @param timeZoneId
   * @return
   */
  public static boolean isValidTimeZone(String timeZoneId) {
    if (isNotNull(timeZoneId)) {
      List<String> supportedTimeZones = Arrays.asList(DateHelper.supportedTimezones);
      return (supportedTimeZones.contains(timeZoneId));
    }
    return false;
  }

  /**
   * Checks if the specified date time is after now (current time).
   *
   * @param dateTime
   * @param timeZone
   * @return
   */
  public static boolean isDateTimeAfterNow(LocalDateTime dateTime, TimeZone timeZone) {
    ZoneId policyZoneId = timeZone.toZoneId();

    ZonedDateTime policyDate = DateHelper.getZonedDateTime(dateTime, timeZone);

    return policyDate.isAfter(ZonedDateTime.now(policyZoneId));
  }

  public static boolean isDateAfterOrEqualsNow(LocalDate date, TimeZone timeZone) {
    ZoneId policyZoneId = timeZone.toZoneId();

    ZonedDateTime policyDate = DateHelper.getZonedDateTime(date, timeZone);

    return !policyDate.toLocalDate().isBefore(ZonedDateTime.now(policyZoneId).toLocalDate());
  }

  /**
   * Checks id the end date time is after start date time
   *
   * @param endDateTime
   * @param startDateTime
   * @return
   */
  public static boolean isAfter(LocalDateTime endDateTime, LocalDateTime startDateTime) {
    if (isNotNull(endDateTime) && isNotNull(startDateTime)) {
      return endDateTime.isAfter(startDateTime);
    }
    return false;
  }

  public static boolean isAfter(LocalTime endTime, LocalTime startTime) {
    if (isNotNull(endTime) && isNotNull(startTime)) {
      return endTime.isAfter(startTime);
    }
    return false;
  }

  public static boolean isBetweenMinAndMaxValues(int[] array, int lowerLimit, int upperLimit) {
    Arrays.sort(array);
    int minValue = array[0];
    int maxValue = array[array.length - 1];

    return (minValue >= lowerLimit && maxValue <= upperLimit);
  }

  public static boolean isElementUnique(int[] array) {
    boolean isValid = true;
    Set<Integer> set = new HashSet<>();
    for (int element : array) {
      if (!set.add(element)) { // Duplicate value found.
        isValid = false;
      }
    }

    return isValid;
  }

  public static List<String[]> isNotOverlapRecurringSchedules(
      List<RecurringScheduleTime> scheduleTimes) {
    List<String[]> overlapDateTimeValidationErrorMsgList = new ArrayList<>();

    if (scheduleTimes != null && !scheduleTimes.isEmpty()) {
      Collections.sort(scheduleTimes);

      for (int firstIndex = 0; firstIndex < scheduleTimes.size(); firstIndex++) {
        for (int secondIndex = firstIndex + 1; secondIndex < scheduleTimes.size(); secondIndex++) {
          RecurringScheduleTime current = scheduleTimes.get(firstIndex);
          RecurringScheduleTime next = scheduleTimes.get(secondIndex);

          if (isStartEndDateOverlapping(current, next)) {
            // both are dayOfWeek
            if (current.hasDayOfWeek() && next.hasDayOfWeek()) {
              // check overlap
              String[] overlapDateTimeValidationErrorMsg =
                  validateTimeOverlapping(
                      current, next, current.getDayOfWeek(), next.getDayOfWeek());
              if (overlapDateTimeValidationErrorMsg != null) {
                overlapDateTimeValidationErrorMsgList.add(overlapDateTimeValidationErrorMsg);
              }
            }

            if (current.hasDayOfMonth() && next.hasDayOfMonth()) {
              // check overlap
              String[] overlapDateTimeValidationErrorMsg =
                  validateTimeOverlapping(
                      current, next, current.getDayOfMonth(), next.getDayOfMonth());
              if (overlapDateTimeValidationErrorMsg != null) {
                overlapDateTimeValidationErrorMsgList.add(overlapDateTimeValidationErrorMsg);
              }
            }
          }
        }
      }
    }
    return overlapDateTimeValidationErrorMsgList;
  }

  private static boolean isStartEndDateOverlapping(
      RecurringScheduleTime current, RecurringScheduleTime next) {
    boolean isOverlapping;

    // NOTE: The Start and End Dates in the schedules are not sorted, so we need to compare dates
    // by  swapping the dates hence calling isSomething twice, once with current and next then with
    // next and current
    isOverlapping =
        isOverlapping(
            current.getStartDate(), current.getEndDate(), next.getStartDate(), next.getEndDate());
    if (!isOverlapping) {
      isOverlapping =
          isOverlapping(
              next.getStartDate(), next.getEndDate(), current.getStartDate(), current.getEndDate());
    }
    return isOverlapping;
  }

  private static boolean isOverlapping(
      LocalDate firstStartDate,
      LocalDate firstEndDate,
      LocalDate secondStartDate,
      LocalDate secondEndDate) {
    boolean isOverlapping =
        (firstStartDate == null && firstEndDate == null)
            || (secondStartDate == null && secondEndDate == null)
            || (firstStartDate == null && secondStartDate == null)
            || (firstEndDate == null && secondEndDate == null);

    if (firstStartDate != null && secondStartDate == null && secondEndDate != null) {
      if (firstStartDate.compareTo(secondEndDate) <= 0) {
        isOverlapping = true;
      }
    }

    if (firstEndDate != null && secondStartDate != null && secondEndDate == null) {
      if (firstEndDate.compareTo(secondStartDate) >= 0) {
        isOverlapping = true;
      }
    }

    if (firstStartDate != null
        && firstEndDate != null
        && secondStartDate != null
        && secondEndDate != null) {
      if (firstStartDate.compareTo(secondStartDate) <= 0
          && firstEndDate.compareTo(secondStartDate) >= 0) {
        isOverlapping = true;
      }
    }

    return isOverlapping;
  }

  private static String[] validateTimeOverlapping(
      RecurringScheduleTime current,
      RecurringScheduleTime next,
      List<Integer> currentDays,
      List<Integer> nextDays) {
    String[] overlapDateTimeValidationErrorMsg = null;

    if (current.getStartTime().compareTo(next.getStartTime()) == 0) {
      if (hasSameElement(currentDays, nextDays)) {
        overlapDateTimeValidationErrorMsg =
            new String[] {
              current.getScheduleIdentifier(),
              "start_time",
              next.getScheduleIdentifier(),
              "start_time"
            };
      }

    } else if (current.getEndTime().compareTo(next.getStartTime()) >= 0) {
      if (hasSameElement(currentDays, nextDays)) {
        overlapDateTimeValidationErrorMsg =
            new String[] {
              current.getScheduleIdentifier(),
              "end_time",
              next.getScheduleIdentifier(),
              "start_time"
            };
      }
    }

    return overlapDateTimeValidationErrorMsg;
  }

  private static boolean hasSameElement(List<Integer> firstList, List<Integer> secondList) {
    for (Integer element : firstList) {
      if (secondList.contains(element)) {
        return true;
      }
    }
    return false;
  }

  /**
   * This method is given a collection of SpecificDateScheduleDateTime (holding the schedule
   * identifier and its start date time and end date time). It traverses through the collection to
   * check if the the date time between different schedules overlap. If there is an overlap then an
   * error message is added to a collection and collection of messages is returned.
   *
   * @param scheduleStartEndTimeList
   * @return - List of date time overlap validation messages
   */
  public static List<String[]> isNotOverlapForSpecificDate(
      List<SpecificDateScheduleDateTime> scheduleStartEndTimeList) {
    List<String[]> overlapDateTimeValidationErrorMsgList = new ArrayList<>();
    if (scheduleStartEndTimeList != null && !scheduleStartEndTimeList.isEmpty()) {

      Collections.sort(scheduleStartEndTimeList);

      for (int index = 0; index < scheduleStartEndTimeList.size() - 1; index++) {
        SpecificDateScheduleDateTime current = scheduleStartEndTimeList.get(index);
        SpecificDateScheduleDateTime next = scheduleStartEndTimeList.get(index + 1);

        // Check for date time overlaps and create a validation error message string array
        if (current.getStartDateTime().compareTo(next.getStartDateTime()) == 0) {

          // startDateTime values are equal, so an overlap. Set up a message for validation error
          String[] overlapDateTimeValidationErrorMsg = {
            current.getScheduleIdentifier(),
            "start_date_time",
            next.getScheduleIdentifier(),
            "start_date_time"
          };
          overlapDateTimeValidationErrorMsgList.add(overlapDateTimeValidationErrorMsg);
        } else if (current.getEndDateTime().compareTo(next.getStartDateTime())
            >= 0) { // current startDateTime was earlier than next startDateTime, so following check

          // endDateTime of current is later than or equal to startDateTime of next. Set up a
          // message for validation error
          String[] overlapDateTimeValidationErrorMsg = {
            current.getScheduleIdentifier(),
            "end_date_time",
            next.getScheduleIdentifier(),
            "start_date_time"
          };
          overlapDateTimeValidationErrorMsgList.add(overlapDateTimeValidationErrorMsg);
        }
      }
    }
    return overlapDateTimeValidationErrorMsgList;
  }
}
