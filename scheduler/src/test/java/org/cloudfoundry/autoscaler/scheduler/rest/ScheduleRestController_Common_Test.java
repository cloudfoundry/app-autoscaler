package org.cloudfoundry.autoscaler.scheduler.rest;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.content;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import java.util.Collections;

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
public class ScheduleRestController_Common_Test {

	@Autowired
	private Scheduler scheduler;

	@Autowired
	private ScheduleDao scheduleDao;

	@Autowired
	MessageBundleResourceHelper messageBundleResourceHelper;

	@Autowired
	private WebApplicationContext wac;
	private MockMvc mockMvc;

	private String appId = TestDataSetupHelper.getAppId_1();

	@Before
	public void beforeTest() throws SchedulerException {
		// Clear previous schedules.
		scheduler.clear();

		mockMvc = MockMvcBuilders.webAppContextSetup(wac).build();

	}

	@After
	@Transactional
	public void afterTest() {
		for (ScheduleEntity entity : scheduleDao.findAllSchedulesByAppId(appId)) {
			scheduleDao.delete(entity);
		}
	}

	@Test
	@Transactional
	public void testCreateSchedule_withNullSchedules_Failure() throws Exception {
		// No schedules - null case
		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, 1);

		schedules.setSpecific_date(null);

		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put(getCreateSchedulePath()).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.invalid.noSchedules",
				"app_id=" + appId);

		assertUserError(resultActions, status().isBadRequest(), errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_withEmptySchedules_Failure() throws Exception {
		// No schedules - Empty case
		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, 1);

		schedules.setSpecific_date(Collections.<ScheduleEntity> emptyList());

		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put(getCreateSchedulePath()).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.invalid.noSchedules",
				"app_id=" + appId);

		assertUserError(resultActions, status().isBadRequest(), errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_withoutAppId_Failure() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, 1);
		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put("/v2/schedules").contentType(MediaType.APPLICATION_JSON).content(content));
		resultActions = mockMvc.perform(get("/v2/schedules").contentType(MediaType.APPLICATION_JSON).content(content));

		resultActions.andExpect(status().isNotFound());

	}

	@Test
	@Transactional
	public void testCreateSchedule_withoutTimeZone_Failure() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, 1);

		schedules.setTimeZone(null);

		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put(getCreateSchedulePath()).contentType(MediaType.APPLICATION_JSON).content(content));

		resultActions.andExpect(status().isBadRequest());
	}

	@Test
	@Transactional
	public void testCreateSchedule_withInvalidTimeZone_Failure() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, 1);

		schedules.setTimeZone(TestDataSetupHelper.getInvalidTimezone());

		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put(getCreateSchedulePath()).contentType(MediaType.APPLICATION_JSON).content(content));

		resultActions.andExpect(status().isBadRequest());
	}

	@Test
	@Transactional
	public void testCreateSchedule_withNullDefaultInstanceMinCount_Failure() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, 1);

		schedules.setInstance_min_count(null);

		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put(getCreateSchedulePath()).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.default.value.invalid", "",
				"instance_min_count");

		assertUserError(resultActions, status().isBadRequest(), errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_withNullDefaultInstanceMaxCountInSpecificDateSchedules_Failure() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, 1);

		schedules.setInstance_max_count(null);

		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put(getCreateSchedulePath()).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.default.value.invalid", "",
				"instance_max_count");

		assertUserError(resultActions, status().isBadRequest(), errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_withNegativeDefaultInstanceMinCount_Failure() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, 1);

		schedules.setInstance_min_count(-1);

		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put(getCreateSchedulePath()).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.default.value.invalid", "",
				"instance_min_count");

		assertUserError(resultActions, status().isBadRequest(), errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_withNegativeDefaultInstanceMaxCount_Failure() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, 1);

		schedules.setInstance_max_count(-1);

		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put(getCreateSchedulePath()).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.default.value.invalid", "",
				"instance_max_count");

		assertUserError(resultActions, status().isBadRequest(), errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_withDefaultInstanceMinCountGreaterThanDefaultInstanceMaxCount_Failure()
			throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, 1);

		Integer tmpInt = schedules.getInstance_max_count();
		schedules.setInstance_max_count(schedules.getInstance_min_count());
		schedules.setInstance_min_count(tmpInt);

		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put(getCreateSchedulePath()).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage(
				"schedule.default.instanceCount.invalid.min.greater", "", "instance_max_count", "instance_min_count");

		assertUserError(resultActions, status().isBadRequest(), errorMessage);
	}

	private String getCreateSchedulePath() {
		return String.format("/v2/schedules/%s", appId);
	}

	private void assertUserError(ResultActions resultActions, ResultMatcher statusCode, String message)
			throws Exception {
		resultActions.andExpect(statusCode);
		resultActions.andExpect(content().contentTypeCompatibleWith(MediaType.APPLICATION_JSON));
		resultActions.andExpect(jsonPath("$").isArray());
		resultActions.andExpect(jsonPath("$[0]").value(Matchers.containsString(message)));
	}
}
