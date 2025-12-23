package org.cloudfoundry.autoscaler.scheduler.misc;

import static org.hamcrest.CoreMatchers.is;
import static org.hamcrest.MatcherAssert.assertThat;

import java.time.LocalDate;
import java.time.LocalDateTime;
import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.time.format.DateTimeFormatter;
import java.time.temporal.TemporalAccessor;
import java.util.ArrayList;
import java.util.List;
import java.util.TimeZone;
import java.util.stream.Stream;
import org.cloudfoundry.autoscaler.scheduler.util.DateHelper;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.Arguments;
import org.junit.jupiter.params.provider.MethodSource;

/**
 * This test class tests the DateHelper class's getZonedDateTime methods.
 * The tests run with different system time zones and policy timezones in a matrix,
 * to make sure that system time zone does not interfere with the date/date time of the policy.
 * <p/>
 * Test Matrix:
 * - System TimeZones: GMT, America/Montreal, Australia/Sydney
 * - Policy TimeZones: GMT, America/Montreal, Australia/Sydney
 * Each test runs for all 9 combinations (3x3 matrix).
 */
class ZonedDateTimeTest {

  private static final String[] SYSTEM_TIME_ZONES = {"GMT", "America/Montreal", "Australia/Sydney"};
  private static final String[] POLICY_TIME_ZONES = {"GMT", "America/Montreal", "Australia/Sydney"};

  private TimeZone originalTimeZone;

  @BeforeEach
  void setUp() {
    originalTimeZone = TimeZone.getDefault();
  }

  @AfterEach
  void tearDown() {
    TimeZone.setDefault(originalTimeZone);
  }

  /**
   * Provides a matrix of system timezone and policy timezone combinations.
   */
  private static Stream<Arguments> provideTimeZoneMatrix() {
    List<Arguments> arguments = new ArrayList<>();
    for (String systemTimeZone : SYSTEM_TIME_ZONES) {
      for (String policyTimeZone : POLICY_TIME_ZONES) {
        arguments.add(Arguments.of(systemTimeZone, policyTimeZone));
      }
    }
    return arguments.stream();
  }

  @ParameterizedTest(
      name =
          "When system timezone is {0} a DateTime in timezone {1} in the policy is correctly read")
  @MethodSource("provideTimeZoneMatrix")
  void testDateTimeForDifferentPolicyTimeZones(String systemTimeZone, String policyTimeZone) {
    // Set the system timezone for this test invocation
    TimeZone.setDefault(TimeZone.getTimeZone(systemTimeZone));

    // Verify that the policy timezone is respected regardless of system timezone
    checkDateHelper_getZonedDateTimeForDateTime(policyTimeZone);
  }

  @ParameterizedTest(
      name = "When system timezone is {0} a Date in timezone {1} in the policy is correctly read")
  @MethodSource("provideTimeZoneMatrix")
  void testDateForDifferentPolicyTimeZones(String systemTimeZone, String policyTimeZone) {
    // Set the system timezone for this test invocation
    TimeZone.setDefault(TimeZone.getTimeZone(systemTimeZone));

    // Verify that the policy timezone is respected regardless of system timezone
    checkDateHelper_getZonedDateTimeForDate(policyTimeZone);
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
