package org.cloudfoundry.autoscaler.scheduler.rest;

import static org.junit.Assert.assertEquals;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.content;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.header;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import javax.transaction.Transactional;

import org.cloudfoundry.autoscaler.scheduler.dao.ScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationScalingSchedules;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
import org.hamcrest.Matchers;
import org.junit.After;
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
import org.springframework.test.web.servlet.ResultMatcher;
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
	private ScheduleDao scheduleDao;

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

	}

	@After
	@Transactional
	public void afterTest() {
		String[] allAppIds = TestDataSetupHelper.getAllAppIds();
		for (String appId : allAppIds) {
			for (ScheduleEntity entity : scheduleDao.findAllSchedulesByAppId(appId)) {
				scheduleDao.delete(entity);
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

		String[] singleAppId = TestDataSetupHelper.generateAppIds(1);
		assertCreateAndGetSchedules(singleAppId, 1);
		assertCreateAndGetSchedules(singleAppId, 5);

		String[] multipleAppIds = TestDataSetupHelper.getAllAppIds();
		assertCreateAndGetSchedules(multipleAppIds, 1);
		assertCreateAndGetSchedules(multipleAppIds, 5);

	}

	private String getCreateSchedulePath(String appId) {
		return String.format("/v2/schedules/%s", appId);
	}

	@Test
	@Transactional
	public void testCreateSchedule_without_appId() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, 1);
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
				.generateSpecificDateSchedulesForScheduleController(appId, 1);

		schedules.setTimeZone(null);

		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put(getCreateSchedulePath(appId)).contentType(MediaType.APPLICATION_JSON).content(content));

		resultActions.andExpect(status().isBadRequest());
	}

	@Test
	@Transactional
	public void testCreateSchedule_invalid_timeZone() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, 1);

		schedules.setTimeZone(TestDataSetupHelper.getInvalidTimezone());

		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put(getCreateSchedulePath(appId)).contentType(MediaType.APPLICATION_JSON).content(content));

		resultActions.andExpect(status().isBadRequest());
	}

	@Test
	@Transactional
	public void testCreateSchedule_without_defaultInstanceMinCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, 1);

		schedules.setInstance_min_count(null);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.default.value.not.specified", "",
				"instance_min_count");

		assertCreateSchedules(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_without_defaultInstanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, 1);

		schedules.setInstance_max_count(null);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.default.value.not.specified", "",
				"instance_max_count");

		assertCreateSchedules(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_negative_defaultInstanceMinCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, 1);
		int instanceMinCount = -1;
		schedules.setInstance_min_count(instanceMinCount);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.default.value.invalid", "",
				"instance_min_count", instanceMinCount);

		assertCreateSchedules(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_negative_defaultInstanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, 1);
		int instanceMaxCount = -1;
		schedules.setInstance_max_count(instanceMaxCount);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.default.value.invalid", "",
				"instance_max_count", instanceMaxCount);

		assertCreateSchedules(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_defaultInstanceMinCount_greater_than_defaultInstanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, 1);

		Integer instanceMinCount = 5;
		Integer instanceMaxCount = 1;
		schedules.setInstance_max_count(instanceMaxCount);
		schedules.setInstance_min_count(instanceMinCount);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage(
				"schedule.default.instanceCount.invalid.min.greater", "", "instance_max_count", instanceMaxCount,
				"instance_min_count", instanceMinCount);

		assertCreateSchedules(appId, content, errorMessage);
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

	private void assertCreateAndGetSchedules(String[] appIds, int expectedSchedulesTobeFound) throws Exception {

		for (String appId : appIds) {
			ResultActions resultActions = callCreateSchedules(appId, expectedSchedulesTobeFound, 0);
			assertCreateScheduleAPI(resultActions);
		}

		for (String appId : appIds) {
			ResultActions resultActions = callGetAllSchedulesByAppId(appId);
			assertSpecificSchedulesFoundEquals(expectedSchedulesTobeFound, appId, resultActions);
		}
		// reset all records for next test.
		afterTest();
	}

	private void assertCreateScheduleAPI(ResultActions resultActions) throws Exception {
		resultActions.andExpect(status().isCreated());
		resultActions.andExpect(header().doesNotExist("Content-type"));
		resultActions.andExpect(content().string(Matchers.isEmptyString()));
	}

	private void assertSpecificSchedulesFoundEquals(int expectedSchedulesTobeFound, String appId,
			ResultActions resultActions) throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules resultSchedules = mapper.readValue(
				resultActions.andReturn().getResponse().getContentAsString(), ApplicationScalingSchedules.class);

		resultActions.andExpect(status().isOk());
		resultActions.andExpect(content().contentTypeCompatibleWith(MediaType.APPLICATION_JSON));
		assertEquals(expectedSchedulesTobeFound, resultSchedules.getSpecific_date().size());
		for (ScheduleEntity entity : resultSchedules.getSpecific_date()) {
			assertEquals(appId, entity.getAppId());
		}
	}

	private void assertCreateSchedules(String appId, String inputContent, String expectedErrorMessage)
			throws Exception {
		ResultActions resultActions = mockMvc.perform(
				put(getCreateSchedulePath(appId)).contentType(MediaType.APPLICATION_JSON).content(inputContent));

		assertUserError(resultActions, status().isBadRequest(), expectedErrorMessage);
	}

	private void assertUserError(ResultActions resultActions, ResultMatcher statusCode, String message)
			throws Exception {
		resultActions.andExpect(statusCode);
		resultActions.andExpect(content().contentTypeCompatibleWith(MediaType.APPLICATION_JSON));
		resultActions.andExpect(jsonPath("$").isArray());
		resultActions.andExpect(jsonPath("$[0]").value(Matchers.containsString(message)));
	}
}
