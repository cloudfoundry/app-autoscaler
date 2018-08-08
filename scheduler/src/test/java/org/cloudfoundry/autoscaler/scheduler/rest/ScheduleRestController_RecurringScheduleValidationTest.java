package org.cloudfoundry.autoscaler.scheduler.rest;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.content;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import java.time.LocalDate;
import java.time.LocalTime;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

import org.cloudfoundry.autoscaler.scheduler.dao.RecurringScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.dao.SpecificDateScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationSchedules;
import org.cloudfoundry.autoscaler.scheduler.util.DateHelper;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleTypeEnum;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataDbUtil;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.TimeZoneTestRule;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
import org.hamcrest.Matchers;
import org.junit.Before;
import org.junit.Rule;
import org.junit.Test;
import org.junit.rules.TestRule;
import org.junit.runner.RunWith;
import org.mockito.Mockito;
import org.quartz.Scheduler;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.context.SpringBootTest.WebEnvironment;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.http.MediaType;
import org.springframework.test.annotation.DirtiesContext;
import org.springframework.test.annotation.DirtiesContext.ClassMode;
import org.springframework.test.context.junit4.SpringRunner;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.ResultActions;
import org.springframework.test.web.servlet.ResultMatcher;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;
import org.springframework.web.context.WebApplicationContext;

import com.fasterxml.jackson.databind.ObjectMapper;

@RunWith(SpringRunner.class)
@SpringBootTest(webEnvironment = WebEnvironment.RANDOM_PORT)
@DirtiesContext(classMode = ClassMode.BEFORE_CLASS)
public class ScheduleRestController_RecurringScheduleValidationTest {

	@Rule
	public TestRule timeZoneRule = new TimeZoneTestRule(new String[] { "America/Los_Angeles", "Australia/Sydney" });

	@MockBean
	private Scheduler scheduler;

	@MockBean
	private SpecificDateScheduleDao specificDateScheduleDao;

	@MockBean
	private RecurringScheduleDao recurringScheduleDao;

	@Autowired
	private MessageBundleResourceHelper messageBundleResourceHelper;

	@Autowired
	private TestDataDbUtil testDataDbUtil;

	@Autowired
	private WebApplicationContext wac;
	private MockMvc mockMvc;

	private String scheduleBeingProcessed = ScheduleTypeEnum.RECURRING.getDescription();

	@Before
	public void beforeTest() throws Exception {
		testDataDbUtil.cleanupData();
		Mockito.reset(scheduler);

		mockMvc = MockMvcBuilders.webAppContextSetup(wac).build();
	}

	@Test
	public void testCreateSchedule_with_startDate() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 5;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);
		LocalDate startDate = TestDataSetupHelper.getZoneDateNow(applicationPolicy.getSchedules().getTimeZone());

		applicationPolicy.getSchedules().getRecurringSchedule().get(0).setStartDate(startDate);

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		assertResponseStatusEquals(resultActions, status().isOk());
	}

	@Test
	public void testCreateSchedule_currentDate_after_startDate() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		LocalDate startDate = TestDataSetupHelper.getZoneDateNow(applicationPolicy.getSchedules().getTimeZone())
				.minusDays(1);
		applicationPolicy.getSchedules().getRecurringSchedule().get(0).setStartDate(startDate);

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.date.invalid.before.current",
				scheduleBeingProcessed + " 0", "start_date", startDate);
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_with_endDate() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 5;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		LocalDate endDate = TestDataSetupHelper.getZoneDateNow(applicationPolicy.getSchedules().getTimeZone())
				.plusDays(7);
		applicationPolicy.getSchedules().getRecurringSchedule().get(0).setEndDate(endDate);

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		assertResponseStatusEquals(resultActions, status().isOk());
	}

	@Test
	public void testCreateSchedule_currentDateTime_after_endDate() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		LocalDate endDate = TestDataSetupHelper.getZoneDateNow(applicationPolicy.getSchedules().getTimeZone())
				.minusDays(1);
		applicationPolicy.getSchedules().getRecurringSchedule().get(0).setEndDate(endDate);

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.date.invalid.before.current",
				scheduleBeingProcessed + " 0", "end_date", endDate);
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_startDate_after_endDate() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		RecurringScheduleEntity entity = applicationPolicy.getSchedules().getRecurringSchedule().get(0);

		// Swap startDate for endDate.
		LocalDate startDate = TestDataSetupHelper.getZoneDateNow(applicationPolicy.getSchedules().getTimeZone())
				.plusDays(2);
		LocalDate endDate = TestDataSetupHelper.getZoneDateNow(applicationPolicy.getSchedules().getTimeZone())
				.plusDays(1);

		entity.setStartDate(startDate);
		entity.setEndDate(endDate);

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.date.invalid.end.before.start",
				scheduleBeingProcessed + " 0", "end_date", entity.getEndDate(), "start_date", entity.getStartDate());
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_without_startTime() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		applicationPolicy.getSchedules().getRecurringSchedule().get(0).setStartTime(null);

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "start_time");
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_without_endTime() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		applicationPolicy.getSchedules().getRecurringSchedule().get(0).setEndTime(null);

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "end_time");
		assertErrorMessage(resultActions, errorMessage);

	}

	@Test
	public void testCreateSchedule_startTime_after_endTime() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		RecurringScheduleEntity entity = applicationPolicy.getSchedules().getRecurringSchedule().get(0);

		// Swap startTime for endTime.
		LocalTime endTime = entity.getStartTime();
		LocalTime startTime = entity.getEndTime();
		entity.setStartTime(startTime);
		entity.setEndTime(endTime);

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.date.invalid.start.after.end",
				scheduleBeingProcessed + " 0", "end_time", DateHelper.convertLocalTimeToString(endTime), "start_time",
				DateHelper.convertLocalTimeToString(startTime));
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_without_instanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		applicationPolicy.getSchedules().getRecurringSchedule().get(0).setInstanceMaxCount(null);

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "instance_max_count");
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_without_instanceMinCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		applicationPolicy.getSchedules().getRecurringSchedule().get(0).setInstanceMinCount(null);

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "instance_min_count");
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_negative_instanceMinCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		int instanceMinCount = -1;
		applicationPolicy.getSchedules().getRecurringSchedule().get(0).setInstanceMinCount(instanceMinCount);

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.invalid",
				scheduleBeingProcessed + " 0", "instance_min_count", instanceMinCount);
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_negative_instanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		int instanceMaxCount = -1;
		applicationPolicy.getSchedules().getRecurringSchedule().get(0).setInstanceMaxCount(instanceMaxCount);

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.invalid",
				scheduleBeingProcessed + " 0", "instance_max_count", instanceMaxCount);
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_instanceMinCount_greater_than_instanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		RecurringScheduleEntity entity = applicationPolicy.getSchedules().getRecurringSchedule().get(0);
		Integer instanceMinCount = 5;
		Integer instanceMaxCount = 4;
		entity.setInstanceMaxCount(instanceMaxCount);
		entity.setInstanceMinCount(instanceMinCount);

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.instanceCount.invalid.min.greater",
				scheduleBeingProcessed + " 0", "instance_max_count", instanceMaxCount, "instance_min_count",
				instanceMinCount);
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_with_initialMinInstanceCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		applicationPolicy.getSchedules().getRecurringSchedule().get(0).setInitialMinInstanceCount(5);

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		assertResponseStatusEquals(resultActions, status().isOk());
	}

	@Test
	public void testCreateSchedule_negative_initialMinInstanceCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);
		Integer initialMinInstanceCount = -1;
		applicationPolicy.getSchedules().getRecurringSchedule().get(0)
				.setInitialMinInstanceCount(initialMinInstanceCount);

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.invalid",
				scheduleBeingProcessed + " 0", "initial_min_instance_count", initialMinInstanceCount);
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_without_dayOfWeek_and_dayOfMonth() throws Exception {
		ResultActions resultActions = getResultActionsForInvalidDayOfWeek(null);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.both.values.not.specified",
				scheduleBeingProcessed + " 0", "day_of_week", "day_of_month");
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_empty_dayOfWeek_and_dayOfMonth() throws Exception {
		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		RecurringScheduleEntity entity = applicationPolicy.getSchedules().getRecurringSchedule().get(0);

		entity.setDaysOfMonth(new int[] {});
		entity.setDaysOfWeek(new int[] {});

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.both.values.not.specified",
				scheduleBeingProcessed + " 0", "day_of_week", "day_of_month");
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_with_dayOfWeek_and_dayOfMonth() throws Exception {
		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		RecurringScheduleEntity entity = applicationPolicy.getSchedules().getRecurringSchedule().get(0);

		entity.setDaysOfMonth(TestDataSetupHelper.generateDayOfMonth());
		entity.setDaysOfWeek(TestDataSetupHelper.generateDayOfWeek());

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.both.values.specified",
				scheduleBeingProcessed + " 0", "day_of_week", "day_of_month");
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_invalid_value_dayOfMonth() throws Exception {
		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.invalid.day",
				scheduleBeingProcessed + " 0", "day_of_month", DateHelper.DAY_OF_MONTH_MINIMUM,
				DateHelper.DAY_OF_MONTH_MAXIMUM);

		ResultActions resultActions = getResultActionsForInvalidDayOfMonth(new int[] { 0 });
		assertErrorMessage(resultActions, errorMessage);

		resultActions = getResultActionsForInvalidDayOfMonth(new int[] { 32 });
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_duplicate_dayOfMonth() throws Exception {
		int[] dayOfMonth = new int[] { 1, 2, 3, 4, 5, 6, 7, 8, 9, 4, 10, 11, 12, 13, 13 };
		ResultActions resultActions = getResultActionsForInvalidDayOfMonth(dayOfMonth);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.not.unique",
				scheduleBeingProcessed + " 0", "day_of_month");
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_invalid_dayOfWeek() throws Exception {
		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.invalid.day",
				scheduleBeingProcessed + " 0", "day_of_week", DateHelper.DAY_OF_WEEK_MINIMUM,
				DateHelper.DAY_OF_WEEK_MAXIMUM);

		ResultActions resultActions = getResultActionsForInvalidDayOfWeek(new int[] { 0 });
		assertErrorMessage(resultActions, errorMessage);
		resultActions = getResultActionsForInvalidDayOfWeek(new int[] { 8 });
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_duplicate_dayOfWeek() throws Exception {
		int[] dayOfWeek = { 2, 3, 4, 5, 6, 5, 7, 7 };
		ResultActions resultActions = getResultActionsForInvalidDayOfWeek(dayOfWeek);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.not.unique",
				scheduleBeingProcessed + " 0", "day_of_week", DateHelper.DAY_OF_WEEK_MINIMUM,
				DateHelper.DAY_OF_WEEK_MAXIMUM);
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_without_recurringSchedules() throws Exception {
		// No schedules - null case
		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		applicationPolicy.getSchedules().setRecurringSchedule(null);

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		assertResponseStatusEquals(resultActions, status().isOk());
	}

	@Test
	public void testCreateSchedule_empty_recurringSchedules() throws Exception {
		// No schedules - Empty case
		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		applicationPolicy.getSchedules().setRecurringSchedule(Collections.emptyList());

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		assertResponseStatusEquals(resultActions, status().isOk());
	}

	@Test
	public void testCreateSchedule_overlapping_startEndTime_with_startEndDate() throws Exception {

		// Overlapping test cases
		ResultActions resultActions = getResultActions(null, null, null, null);
		assertOverlapDateErrorMessage(resultActions);
		resultActions = getResultActions("9999-01-01", null, null, null);
		assertOverlapDateErrorMessage(resultActions);
		resultActions = getResultActions(null, "9999-01-01", null, null);
		assertOverlapDateErrorMessage(resultActions);
		resultActions = getResultActions(null, null, "9999-01-01", null);
		assertOverlapDateErrorMessage(resultActions);
		resultActions = getResultActions(null, null, null, "9999-01-01");
		assertOverlapDateErrorMessage(resultActions);
		resultActions = getResultActions("9999-01-01", "9999-01-01", null, null);
		assertOverlapDateErrorMessage(resultActions);
		resultActions = getResultActions("9999-01-01", null, "9999-01-01", null);
		assertOverlapDateErrorMessage(resultActions);
		resultActions = getResultActions("9999-01-01", null, null, "9999-01-01");
		assertOverlapDateErrorMessage(resultActions);
		resultActions = getResultActions(null, "9999-01-01", "9999-01-01", null);
		assertOverlapDateErrorMessage(resultActions);
		resultActions = getResultActions(null, "9999-01-01", null, "9999-01-01");
		assertOverlapDateErrorMessage(resultActions);
		resultActions = getResultActions(null, null, "9999-01-01", "9999-01-01");
		assertOverlapDateErrorMessage(resultActions);
		resultActions = getResultActions("9999-01-01", "9999-01-01", "9999-01-01", null);
		assertOverlapDateErrorMessage(resultActions);
		resultActions = getResultActions("9999-01-01", "9999-01-01", null, "9999-01-01");
		assertOverlapDateErrorMessage(resultActions);
		resultActions = getResultActions("9999-01-01", null, "9999-01-01", "9999-01-01");
		assertOverlapDateErrorMessage(resultActions);
		resultActions = getResultActions(null, "9999-01-01", "9999-01-01", "9999-01-01");
		assertOverlapDateErrorMessage(resultActions);
		resultActions = getResultActions("9999-01-01", "9999-12-01", "9999-01-05", "9999-12-05");
		assertOverlapDateErrorMessage(resultActions);
		resultActions = getResultActions("9999-01-01", "9999-12-01", "9999-01-01", "9999-12-01");
		assertOverlapDateErrorMessage(resultActions);
		resultActions = getResultActions("9999-01-01", "9999-12-01", "9999-01-01", "9999-12-01");
		assertOverlapDateErrorMessage(resultActions);
		resultActions = getResultActions("9999-01-01", "9999-12-01", "9998-12-01", "9999-10-01");
		assertOverlapDateErrorMessage(resultActions);

		// Not overlapping test cases
		resultActions = getResultActions("9999-01-05", null, null, "9999-01-04");
		assertResponseStatusEquals(resultActions, status().isOk());
		resultActions = getResultActions(null, "9999-01-04", "9999-01-05", null);
		assertResponseStatusEquals(resultActions, status().isOk());
		resultActions = getResultActions("9999-01-01", "9999-12-01", "9999-12-05", null);
		assertResponseStatusEquals(resultActions, status().isOk());
		resultActions = getResultActions("9999-01-05", "9999-12-01", null, "9999-01-01");
		assertResponseStatusEquals(resultActions, status().isOk());
		resultActions = getResultActions("9999-01-01", null, "9998-01-05", "9998-12-31");
		assertResponseStatusEquals(resultActions, status().isOk());
		resultActions = getResultActions(null, "9999-01-05", "9999-01-06", "9999-12-05");
		assertResponseStatusEquals(resultActions, status().isOk());
		resultActions = getResultActions("9998-01-01", "9998-12-31", "9999-01-01", "9999-12-31");
		assertResponseStatusEquals(resultActions, status().isOk());
		resultActions = getResultActions("9999-01-01", "9999-12-01", "9998-01-01", "9998-12-31");
		assertResponseStatusEquals(resultActions, status().isOk());
	}

	@Test
	public void testCreateSchedule_overlapping_startEndTime_and_overlapping_dayOfWeek() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 2;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		// Overlap recurring applicationPolicy.getSchedules().
		RecurringScheduleEntity firstEntity = applicationPolicy.getSchedules().getRecurringSchedule().get(0);
		RecurringScheduleEntity secondEntity = applicationPolicy.getSchedules().getRecurringSchedule().get(1);
		secondEntity.setStartTime(firstEntity.getEndTime());

		firstEntity.setDaysOfWeek(TestDataSetupHelper.generateDayOfWeek());
		firstEntity.setDaysOfMonth(null);

		secondEntity.setDaysOfWeek(firstEntity.getDaysOfWeek());
		secondEntity.setDaysOfMonth(null);

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		assertOverlapDateErrorMessage(resultActions);
	}

	@Test
	public void testCreateSchedule_overlapping_startEndTime_and_overlapping_dayOfMonth() throws Exception {
		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 2;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		// Overlap recurring applicationPolicy.getSchedules().
		RecurringScheduleEntity firstEntity = applicationPolicy.getSchedules().getRecurringSchedule().get(0);
		RecurringScheduleEntity secondEntity = applicationPolicy.getSchedules().getRecurringSchedule().get(1);
		secondEntity.setStartTime(firstEntity.getEndTime());

		firstEntity.setDaysOfWeek(null);
		firstEntity.setDaysOfMonth(TestDataSetupHelper.generateDayOfMonth());

		secondEntity.setDaysOfWeek(null);
		secondEntity.setDaysOfMonth(firstEntity.getDaysOfMonth());

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		assertOverlapDateErrorMessage(resultActions);
	}

	@Test
	public void testCreateSchedule_overlapping_dayOfMonth_and_dayOfWeek() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 4;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		// Overlap recurring applicationPolicy.getSchedules().
		// Schedule 1 end date, end time and Schedule 2 start date, start time
		// are overlapping.
		// Schedules 3 and 4 is overlap with start date and start time.
		RecurringScheduleEntity firstEntity = applicationPolicy.getSchedules().getRecurringSchedule().get(0);
		RecurringScheduleEntity secondEntity = applicationPolicy.getSchedules().getRecurringSchedule().get(1);
		secondEntity.setStartDate(firstEntity.getEndDate());
		secondEntity.setStartTime(firstEntity.getEndTime());

		firstEntity.setDaysOfWeek(null);
		firstEntity.setDaysOfMonth(new int[] { 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
				21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31 });
		secondEntity.setDaysOfWeek(new int[] { 1, 2, 3, 4, 5, 6, 7 });
		secondEntity.setDaysOfMonth(null);

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		assertResponseStatusEquals(resultActions, status().isOk());
	}

	@Test
	public void testCreateSchedule_overlapping_multipleSchedules() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 4;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		// Overlap recurring applicationPolicy.getSchedules().
		// Schedule 1 end date, end time and Schedule 2 start date, start time
		// are overlapping.
		// Schedules 3 and 4 is overlap with start date and start time.
		RecurringScheduleEntity firstEntity = applicationPolicy.getSchedules().getRecurringSchedule().get(0);
		RecurringScheduleEntity secondEntity = applicationPolicy.getSchedules().getRecurringSchedule().get(1);
		secondEntity.setStartDate(firstEntity.getEndDate());
		secondEntity.setStartTime(firstEntity.getEndTime());

		firstEntity.setDaysOfWeek(null);
		firstEntity.setDaysOfMonth(TestDataSetupHelper.generateDayOfMonth());
		secondEntity.setDaysOfWeek(null);
		secondEntity.setDaysOfMonth(firstEntity.getDaysOfMonth());

		RecurringScheduleEntity thirdEntity = applicationPolicy.getSchedules().getRecurringSchedule().get(2);
		RecurringScheduleEntity forthEntity = applicationPolicy.getSchedules().getRecurringSchedule().get(3);
		forthEntity.setStartDate(thirdEntity.getStartDate());
		forthEntity.setStartTime(thirdEntity.getStartTime());

		thirdEntity.setDaysOfWeek(TestDataSetupHelper.generateDayOfWeek());
		thirdEntity.setDaysOfMonth(null);

		forthEntity.setDaysOfWeek(thirdEntity.getDaysOfWeek());
		forthEntity.setDaysOfMonth(null);

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		List<String> messages = new ArrayList<>();
		messages.add(messageBundleResourceHelper.lookupMessage("schedule.date.overlap", scheduleBeingProcessed + " 0",
				"end_time", scheduleBeingProcessed + " 1", "start_time"));
		messages.add(messageBundleResourceHelper.lookupMessage("schedule.date.overlap", scheduleBeingProcessed + " 2",
				"start_time", scheduleBeingProcessed + " 3", "start_time"));

		assertErrorMessage(resultActions, messages.toArray(new String[0]));
	}

	@Test
	public void testCreateSchedule_without_startEndTime_instanceMaxMinCount() throws Exception {
		// schedules - no parameters.
		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		RecurringScheduleEntity entity = applicationPolicy.getSchedules().getRecurringSchedule().get(0);
		entity.setInstanceMinCount(null);
		entity.setInstanceMaxCount(null);
		entity.setStartTime(null);
		entity.setEndTime(null);

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		List<String> messages = new ArrayList<>();
		messages.add(messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "start_time"));
		messages.add(messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "end_time"));
		messages.add(messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "instance_max_count"));
		messages.add(messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "instance_min_count"));

		assertErrorMessage(resultActions, messages.toArray(new String[0]));
	}

	private ResultActions getResultActions(String firstStartDateStr, String firstEndDateStr, String secondStartDateStr,
			String secondEndDateStr) throws Exception {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		String content = TestDataSetupHelper.generateJsonForOverlappingRecurringScheduleWithStartEndDate(
				firstStartDateStr, firstEndDateStr, secondStartDateStr, secondEndDateStr);

		return mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));
	}

	private ResultActions getResultActionsForInvalidDayOfWeek(int[] dayOfWeek) throws Exception {
		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		RecurringScheduleEntity entity = applicationPolicy.getSchedules().getRecurringSchedule().get(0);

		entity.setDaysOfMonth(null);
		entity.setDaysOfWeek(dayOfWeek);

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		return mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));
	}

	private ResultActions getResultActionsForInvalidDayOfMonth(int[] dayOfMonth) throws Exception {
		ObjectMapper mapper = new ObjectMapper();
		int noOfRecurringSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(0,
				noOfRecurringSchedulesToSetUp);

		RecurringScheduleEntity entity = applicationPolicy.getSchedules().getRecurringSchedule().get(0);

		entity.setDaysOfMonth(dayOfMonth);
		entity.setDaysOfWeek(null);

		String content = mapper.writeValueAsString(applicationPolicy);
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		String guid = TestDataSetupHelper.generateGuid();
		return mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));
	}

	private void assertOverlapDateErrorMessage(ResultActions resultActions) throws Exception {
		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.date.overlap",
				scheduleBeingProcessed + " 0", "end_time", scheduleBeingProcessed + " 1", "start_time");
		assertErrorMessage(resultActions, errorMessage);
	}

	private void assertResponseStatusEquals(ResultActions resultActions, ResultMatcher status) throws Exception {
		resultActions.andExpect(content().string(""));
		resultActions.andExpect(status);
	}

	private void assertErrorMessage(ResultActions resultActions, String... expectedErrorMessages) throws Exception {
		resultActions.andExpect(jsonPath("$").value(Matchers.containsInAnyOrder(expectedErrorMessages)));
		resultActions.andExpect(jsonPath("$").isArray());
		resultActions.andExpect(content().contentTypeCompatibleWith(MediaType.APPLICATION_JSON));
		resultActions.andExpect(status().isBadRequest());
	}

}
