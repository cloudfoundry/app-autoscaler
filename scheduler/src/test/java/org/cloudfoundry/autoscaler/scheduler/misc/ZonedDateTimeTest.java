package org.cloudfoundry.autoscaler.scheduler.misc;

import static org.hamcrest.CoreMatchers.is;
import static org.hamcrest.MatcherAssert.assertThat;

import java.time.LocalDate;
import java.time.LocalDateTime;
import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.time.format.DateTimeFormatter;
import java.time.temporal.TemporalAccessor;
import java.util.TimeZone;
import org.cloudfoundry.autoscaler.scheduler.util.DateHelper;
import org.cloudfoundry.autoscaler.scheduler.util.TimeZoneTestRule;
import org.junit.Rule;
import org.junit.Test;
import org.junit.rules.TestRule;
import org.junit.runner.RunWith;
import org.junit.runners.JUnit4;

/**
 * This test class tests the DateHelper class's getZonedDateTime methods. The rule changes the
 * system time zone to specified ones and the tests are run for different policy timezones, so as to
 * make sure that system time zone does not interfere with the date/ date time of the policy.
 */
@RunWith(JUnit4.class)
public class ZonedDateTimeTest {

  @Rule
  public TestRule systemTimeZoneRule =
      new TimeZoneTestRule(new String[] {"GMT", "America/Montreal", "Australia/Sydney"});

  @Test
  public void testDateTimeForDifferentPolicyTimeZones() {
    // The following methods are called with specified policy timezones. Idea is to check
    // when the system time zone is different from the policy's time zone, then it does not
    // impact the schedule's datetime.
    checkDateHelper_getZonedDateTimeForDateTime("GMT");
    checkDateHelper_getZonedDateTimeForDateTime("America/Montreal");
    checkDateHelper_getZonedDateTimeForDateTime("Australia/Sydney");
  }

  @Test
  public void testDateForDifferentPolicyTimeZones() {
    // The following methods are called with specified policy timezones. Idea is to check
    // when the system time zone is different from the policy's time zone, then it does not
    // impact the schedule's date.
    checkDateHelper_getZonedDateTimeForDate("GMT");
    checkDateHelper_getZonedDateTimeForDate("America/Montreal");
    checkDateHelper_getZonedDateTimeForDate("Australia/Sydney");
  }

  private void checkDateHelper_getZonedDateTimeForDateTime(String policyTimeZone) {
    ZonedDateTime convertedDate =
        getZonedDateTimeForDateTime("2017-03-12T01:00", TimeZone.getTimeZone(policyTimeZone));
    assertThat(
        "Zoned date time is as expected when it is not DST in Montreal",
        convertedDate,
        is(parseZonedDateTime("2017-03-12T01:00Z[" + policyTimeZone + "]")));

    convertedDate =
        getZonedDateTimeForDateTime("2017-03-12T02:00", TimeZone.getTimeZone(policyTimeZone));
    assertThat(
        "Zoned date time is as expected at the time of DST start in Montreal",
        convertedDate,
        is(parseZonedDateTime("2017-03-12T02:00Z[" + policyTimeZone + "]")));

    convertedDate =
        getZonedDateTimeForDateTime("2017-03-12T03:00", TimeZone.getTimeZone(policyTimeZone));
    assertThat(
        "Zoned date time is as expected when DST in Montreal",
        convertedDate,
        is(parseZonedDateTime("2017-03-12T03:00Z[" + policyTimeZone + "]")));

    convertedDate =
        getZonedDateTimeForDateTime("2017-11-05T01:00", TimeZone.getTimeZone(policyTimeZone));
    assertThat(
        "Zoned date time is as expected at the time of DST end in Montreal",
        convertedDate,
        is(parseZonedDateTime("2017-11-05T01:00Z[" + policyTimeZone + "]")));

    convertedDate =
        getZonedDateTimeForDateTime("2017-11-05T02:00", TimeZone.getTimeZone(policyTimeZone));
    assertThat(
        "Zoned date time is as expected just after DST ended in Montreal",
        convertedDate,
        is(parseZonedDateTime("2017-11-05T02:00Z[" + policyTimeZone + "]")));

    convertedDate =
        getZonedDateTimeForDateTime("2017-10-01T01:00", TimeZone.getTimeZone(policyTimeZone));
    assertThat(
        "Zoned date time is as expected when it is not DST in Sydney",
        convertedDate,
        is(parseZonedDateTime("2017-10-01T01:00Z[" + policyTimeZone + "]")));

    convertedDate =
        getZonedDateTimeForDateTime("2017-10-01T02:00", TimeZone.getTimeZone(policyTimeZone));
    assertThat(
        "Zoned date time is as expected at the time of DST start in Sydney",
        convertedDate,
        is(parseZonedDateTime("2017-10-01T02:00Z[" + policyTimeZone + "]")));

    convertedDate =
        getZonedDateTimeForDateTime("2017-10-01T03:00", TimeZone.getTimeZone(policyTimeZone));
    assertThat(
        "Zoned date time is as expected when DST in Sydney",
        convertedDate,
        is(parseZonedDateTime("2017-10-01T03:00Z[" + policyTimeZone + "]")));

    convertedDate =
        getZonedDateTimeForDateTime("2017-04-02T02:00", TimeZone.getTimeZone(policyTimeZone));
    assertThat(
        "Zoned date time is as expected at the time of DST end in Sydney",
        convertedDate,
        is(parseZonedDateTime("2017-04-02T02:00Z[" + policyTimeZone + "]")));

    convertedDate =
        getZonedDateTimeForDateTime("2017-04-02T03:00", TimeZone.getTimeZone(policyTimeZone));
    assertThat(
        "Zoned date time is as expected just after DST ended in Sydney",
        convertedDate,
        is(parseZonedDateTime("2017-04-02T03:00Z[" + policyTimeZone + "]")));
  }

  private void checkDateHelper_getZonedDateTimeForDate(String policyTimeZone) {
    // Test the date is correct for different system timezones and different policy timezone.
    ZonedDateTime convertedDate =
        getZonedDateTimeForDate("2017-03-12", TimeZone.getTimeZone(policyTimeZone));
    assertThat(convertedDate, is(parseZonedDateTime("2017-03-12T00:00Z[" + policyTimeZone + "]")));
  }

  private ZonedDateTime getZonedDateTimeForDateTime(String inputDateTimeStr, TimeZone timeZone) {
    DateTimeFormatter dateTimeFormatter = DateTimeFormatter.ofPattern(DateHelper.DATE_TIME_FORMAT);
    LocalDateTime dateTime = LocalDateTime.parse(inputDateTimeStr, dateTimeFormatter);
    return DateHelper.getZonedDateTime(dateTime, timeZone);
  }

  private ZonedDateTime getZonedDateTimeForDate(String inputDateStr, TimeZone timeZone) {
    DateTimeFormatter dateTimeFormatter = DateTimeFormatter.ofPattern(DateHelper.DATE_FORMAT);
    LocalDate date = LocalDate.parse(inputDateStr, dateTimeFormatter);
    return DateHelper.getZonedDateTime(date, timeZone);
  }

  private ZonedDateTime parseZonedDateTime(String policyTimeZone) {
    // https://stackoverflow.com/questions/56255020/zoneddatetime-change-behavior-jdk-8-11
    TemporalAccessor parsed = DateTimeFormatter.ISO_ZONED_DATE_TIME.parse(policyTimeZone);
    LocalDateTime dateTime = LocalDateTime.from(parsed);
    ZoneId zone = ZoneId.from(parsed);
    return dateTime.atZone(zone);
  }
}
