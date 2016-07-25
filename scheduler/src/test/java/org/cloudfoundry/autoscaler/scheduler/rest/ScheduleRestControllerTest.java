package org.cloudfoundry.autoscaler.scheduler.rest;

import static org.junit.Assert.assertEquals;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.content;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.header;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import java.util.ArrayList;
import java.util.List;

import javax.transaction.Transactional;

import org.cloudfoundry.autoscaler.scheduler.dao.SpecificDateScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationScalingSchedules;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
import org.hamcrest.Matchers;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.quartz.Scheduler;
import org.quartz.SchedulerException;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.context.SpringBootTest.WebEnvironment;
import org.springframework.http.MediaType;
import org.springframework.test.annotation.DirtiesContext;
import org.springframework.test.annotation.DirtiesContext.ClassMode;
import org.springframework.test.context.junit4.SpringRunner;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.ResultActions;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;
import org.springframework.web.context.WebApplicationContext;

import com.fasterxml.jackson.databind.ObjectMapper;

/**
 * 
 *
 */
@RunWith(SpringRunner.class)
@SpringBootTest(webEnvironment = WebEnvironment.RANDOM_PORT)
@DirtiesContext(classMode = ClassMode.BEFORE_EACH_TEST_METHOD)
public class ScheduleRestControllerTest {

	@Autowired
	private Scheduler scheduler;

	@Autowired
	private SpecificDateScheduleDao specificDateScheduleDao;

	@Autowired
	MessageBundleResourceHelper messageBundleResourceHelper;

	@Autowired
	private WebApplicationContext wac;
	private MockMvc mockMvc;

	String appId = TestDataSetupHelper.generateAppIds(1)[0];

	@Before
	public void beforeTest() throws SchedulerException {
		// Clear previous schedules.
		scheduler.clear();

		mockMvc = MockMvcBuilders.webAppContextSetup(wac).build();
		removeAllRecordsFromDatabase();
	}

	@Transactional
	public void removeAllRecordsFromDatabase() {
		List<String> allAppIds = TestDataSetupHelper.getAllGeneratedAppIds();
		for (String appId : allAppIds) {
			for (SpecificDateScheduleEntity entity : specificDateScheduleDao
					.findAllSpecificDateSchedulesByAppId(appId)) {
				specificDateScheduleDao.delete(entity);
			}
		}
	}

	@Test
	@Transactional
	public void testGetAllSchedule_with_no_schedules() throws Exception {
		ResultActions resultActions = callGetAllSchedulesByAppId(appId);

		resultActions.andExpect(status().isNotFound());
		resultActions.andExpect(header().doesNotExist("Content-type"));
		resultActions.andExpect(content().string(Matchers.isEmptyString()));
	}

	@Test
	@Transactional
	public void testCreateAndGetSchedules() throws Exception {
		// Test multiple applications each having single specific date schedule, no recurring schedule
		String[] multipleAppIds = TestDataSetupHelper.generateAppIds(5);
		assertCreateAndGetSchedules(multipleAppIds, 1, 0);

		// Test multiple applications each having multiple specific date schedules, no recurring schedule
		multipleAppIds = TestDataSetupHelper.generateAppIds(5);
		assertCreateAndGetSchedules(multipleAppIds, 5, 0);

		// Test multiple applications each having single recurring schedule, no specific date schedules
		multipleAppIds = TestDataSetupHelper.generateAppIds(5);
		assertCreateAndGetSchedules(multipleAppIds, 0, 1);

		// Test multiple applications each having multiple recurring schedule, no specific date schedules
		multipleAppIds = TestDataSetupHelper.generateAppIds(5);
		assertCreateAndGetSchedules(multipleAppIds, 0, 5);

		// Test multiple applications each having multiple specific date and multiple recurring schedule 
		multipleAppIds = TestDataSetupHelper.generateAppIds(5);
		assertCreateAndGetSchedules(multipleAppIds, 5, 5);
	}

	@Test
	@Transactional
	public void testCreateSchedule_already_existing_schedule_for_appId() throws Exception {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		// Create one specific date schedule and two recurring schedules for the application.
		callCreateSchedules(appId, 1, 2);

		// Create two specific date schedules and one recurring schedules for the same application.
		ResultActions resultActions = callCreateSchedules(appId, 2, 1);
		resultActions.andExpect(status().isConflict());
	}

	@Test
	@Transactional
	public void testCreateSchedule_without_appId() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedulesForRestApi(1, 0);
		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put("/v2/schedules").contentType(MediaType.APPLICATION_JSON).content(content));

		resultActions.andExpect(status().isNotFound());

	}

	@Test
	@Transactional
	public void testCreateSchedule_without_timeZone() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedulesForRestApi(1, 0);

		schedules.setTimeZone(null);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.value.not.specified.timezone",
				"timeZone");

		assertErrorMessages(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_empty_timeZone() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedulesForRestApi(1, 0);

		schedules.setTimeZone("");

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.value.not.specified.timezone",
				"timeZone");

		assertErrorMessages(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_invalid_timeZone() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedulesForRestApi(1, 0);

		schedules.setTimeZone(TestDataSetupHelper.getInvalidTimezone());

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.invalid.timezone", "timeZone");

		assertErrorMessages(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_without_defaultInstanceMinCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedulesForRestApi(1, 0);

		schedules.setInstance_min_count(null);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.default.value.not.specified",
				"instance_min_count");

		assertErrorMessages(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_without_defaultInstanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedulesForRestApi(1, 0);

		schedules.setInstance_max_count(null);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.default.value.not.specified",
				"instance_max_count");

		assertErrorMessages(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_negative_defaultInstanceMinCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedulesForRestApi(1, 0);
		int instanceMinCount = -1;
		schedules.setInstance_min_count(instanceMinCount);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.default.value.invalid",
				"instance_min_count", instanceMinCount);

		assertErrorMessages(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_negative_defaultInstanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedulesForRestApi(1, 0);
		int instanceMaxCount = -1;
		schedules.setInstance_max_count(instanceMaxCount);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.default.value.invalid",
				"instance_max_count", instanceMaxCount);

		assertErrorMessages(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_defaultInstanceMinCount_greater_than_defaultInstanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedulesForRestApi(1, 0);

		Integer instanceMinCount = 5;
		Integer instanceMaxCount = 1;
		schedules.setInstance_max_count(instanceMaxCount);
		schedules.setInstance_min_count(instanceMinCount);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage(
				"data.default.instanceCount.invalid.min.greater", "instance_max_count", instanceMaxCount,
				"instance_min_count", instanceMinCount);

		assertErrorMessages(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_without_instanceMaxAndMinCount_timeZone() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedulesForRestApi(1, 0);

		schedules.setInstance_max_count(null);
		schedules.setInstance_min_count(null);
		schedules.setTimeZone(null);

		String content = mapper.writeValueAsString(schedules);

		List<String> messages = new ArrayList<>();
		messages.add(
				messageBundleResourceHelper.lookupMessage("data.default.value.not.specified", "instance_min_count"));
		messages.add(
				messageBundleResourceHelper.lookupMessage("data.default.value.not.specified", "instance_max_count"));
		messages.add(messageBundleResourceHelper.lookupMessage("data.value.not.specified.timezone", "timeZone"));

		assertErrorMessages(appId, content, messages.toArray(new String[0]));
	}

	@Test
	@Transactional
	public void testCreateSchedule_multiple_error() throws Exception {
		// Should be individual each test.

		testCreateSchedule_negative_defaultInstanceMaxCount();

		testCreateSchedule_without_defaultInstanceMinCount();

		testCreateSchedule_without_defaultInstanceMaxCount();

		testCreateSchedule_defaultInstanceMinCount_greater_than_defaultInstanceMaxCount();
	}

	private String getCreateSchedulePath(String appId) {
		return String.format("/v2/schedules/%s", appId);
	}

	private ResultActions callCreateSchedules(String appId, int noOfSpecificDateSchedulesToSetUp,
			int noOfRecurringSchedulesToSetUp) throws Exception {
		String content = TestDataSetupHelper.generateJsonSchedule(appId, noOfSpecificDateSchedulesToSetUp,
				noOfRecurringSchedulesToSetUp);

		return mockMvc
				.perform(put(getCreateSchedulePath(appId)).contentType(MediaType.APPLICATION_JSON).content(content));

	}

	private ResultActions callGetAllSchedulesByAppId(String appId) throws Exception {

		return mockMvc.perform(get(getCreateSchedulePath(appId)).accept(MediaType.APPLICATION_JSON));

	}

	private void assertCreateAndGetSchedules(String[] appIds, int expectedSpecificDateSchedulesTobeFound,
			int expectedRecurringScheduleTobeFound) throws Exception {

		for (String appId : appIds) {
			ResultActions resultActions = callCreateSchedules(appId, expectedSpecificDateSchedulesTobeFound,
					expectedRecurringScheduleTobeFound);
			assertCreateScheduleAPI(resultActions);
		}

		for (String appId : appIds) {
			ResultActions resultActions = callGetAllSchedulesByAppId(appId);
			assertSchedulesFoundEquals(expectedSpecificDateSchedulesTobeFound, expectedRecurringScheduleTobeFound, appId,
					resultActions);
		}
	}

	private void assertCreateScheduleAPI(ResultActions resultActions) throws Exception {
		resultActions.andExpect(status().isCreated());
		resultActions.andExpect(header().doesNotExist("Content-type"));
		resultActions.andExpect(content().string(Matchers.isEmptyString()));
	}

	private void assertSchedulesFoundEquals(int expectedSpecificDateSchedulesTobeFound, int expectedRecurringSchedulesTobeFound,
			String appId, ResultActions resultActions) throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules resultSchedules = mapper.readValue(
				resultActions.andReturn().getResponse().getContentAsString(), ApplicationScalingSchedules.class);

		resultActions.andExpect(status().isOk());
		resultActions.andExpect(content().contentTypeCompatibleWith(MediaType.APPLICATION_JSON));

		assertSpecificDateScheduleFoundEquals(expectedSpecificDateSchedulesTobeFound, appId, resultSchedules.getSpecific_date());
		assertRecurringDateScheduleFoundEquals(expectedRecurringSchedulesTobeFound, appId,
				resultSchedules.getRecurring_schedule());
	}

	private void assertSpecificDateScheduleFoundEquals(int expectedSchedulesTobeFound, String expectedAppId,
			List<SpecificDateScheduleEntity> specificDateScheduls) {
		if (specificDateScheduls == null) {
			assertEquals(expectedSchedulesTobeFound, 0);
		} else {
			assertEquals(expectedSchedulesTobeFound, specificDateScheduls.size());
			for (ScheduleEntity entity : specificDateScheduls) {
				assertEquals(expectedAppId, entity.getAppId());
			}
		}
	}

	private void assertRecurringDateScheduleFoundEquals(int expectedRecurringSchedulesTobeFound, String expectedAppId,
			List<RecurringScheduleEntity> recurring_schedule) {
		if (recurring_schedule == null) {
			assertEquals(expectedRecurringSchedulesTobeFound, 0);
		} else {
			assertEquals(expectedRecurringSchedulesTobeFound, recurring_schedule.size());
			for (ScheduleEntity entity : recurring_schedule) {
				assertEquals(expectedAppId, entity.getAppId());
			}
		}
	
	}

	private void assertErrorMessages(String appId, String inputContent, String... expectedErrorMessages)
			throws Exception {
		ResultActions resultActions = mockMvc.perform(
				put(getCreateSchedulePath(appId)).contentType(MediaType.APPLICATION_JSON).content(inputContent));

		resultActions.andExpect(status().isBadRequest());
		resultActions.andExpect(content().contentTypeCompatibleWith(MediaType.APPLICATION_JSON));
		resultActions.andExpect(jsonPath("$").isArray());
		resultActions.andExpect(jsonPath("$").value(Matchers.containsInAnyOrder(expectedErrorMessages)));
	}
}
