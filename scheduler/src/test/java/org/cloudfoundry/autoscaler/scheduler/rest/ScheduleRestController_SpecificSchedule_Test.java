package org.cloudfoundry.autoscaler.scheduler.rest;

import static org.junit.Assert.assertEquals;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.content;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import java.sql.Time;

import javax.transaction.Transactional;

import org.cloudfoundry.autoscaler.scheduler.dao.ScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationScalingSchedules;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleTypeEnum;
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
public class ScheduleRestController_SpecificSchedule_Test {

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
	private String scheduleBeingProcessed = ScheduleTypeEnum.SPECIFIC_DATE.getDescription();

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
	public void testGetSchedule_01() throws Exception {
		// Expected no schedule
		
		ResultActions resultActions = getResultActionWithGetSchedule();

		ObjectMapper mapper = new ObjectMapper();
		resultActions.andExpect(status().isOk());
		ApplicationScalingSchedules resultSchedules = mapper.readValue(
				resultActions.andReturn().getResponse().getContentAsString(), ApplicationScalingSchedules.class);
		assertEquals(0, resultSchedules.getSpecific_date().size());
	}
	
	@Test
	@Transactional
	public void testGetSchedule_02() throws Exception {
		// Expected one schedule
		ResultActions resultActions = getResultActionWithCreateSchedule(1, 0);

		resultActions.andExpect(status().isCreated());

		resultActions = getResultActionWithGetSchedule();

		ObjectMapper mapper = new ObjectMapper();
		resultActions.andExpect(status().isOk());
		ApplicationScalingSchedules resultSchedules = mapper.readValue(
				resultActions.andReturn().getResponse().getContentAsString(), ApplicationScalingSchedules.class);
		assertEquals(1, resultSchedules.getSpecific_date().size());
	}
	
	@Test
	@Transactional
	public void testGetSchedule_03() throws Exception {
		// Expected multiple schedules	
		int noOfSpecificDateSchedulesToSetUp = 4;
		ResultActions resultActions = getResultActionWithCreateSchedule(noOfSpecificDateSchedulesToSetUp, 0);

		resultActions.andExpect(status().isCreated());

		resultActions = getResultActionWithGetSchedule();

		ObjectMapper mapper = new ObjectMapper();
		resultActions.andExpect(status().isOk());
		ApplicationScalingSchedules resultSchedules = mapper.readValue(
				resultActions.andReturn().getResponse().getContentAsString(), ApplicationScalingSchedules.class);
		assertEquals(noOfSpecificDateSchedulesToSetUp, resultSchedules.getSpecific_date().size());
	}

	@Test
	@Transactional
	public void testCreateSchedule_04() throws Exception {
		// Create one schedule
		ResultActions resultActions = getResultActionWithCreateSchedule(1, 0);

		resultActions.andExpect(status().isCreated());
	}

	@Test
	@Transactional
	public void testCreateSchedule_05() throws Exception {
		// Create multiple schedules
		
		int noOfSpecificDateSchedulesToSetUp = 4;
		ResultActions resultActions = getResultActionWithCreateSchedule(noOfSpecificDateSchedulesToSetUp, 0);

		resultActions.andExpect(status().isCreated());
	}

	@Test
	@Transactional
	public void testCreateScheduleWithoutStartDate_Failure_06() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		schedules.getSpecific_date().get(0).setStartDate(null);

		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put(getCreateSchedulePath()).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.null",
				scheduleBeingProcessed + " 1", "start_date");

		assertUserError(resultActions, status().isBadRequest(), errorMessage);
	}

	@Test
	@Transactional
	public void testCreateScheduleWithoutEndDate_Failure_07() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		schedules.getSpecific_date().get(0).setEndDate(null);

		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put(getCreateSchedulePath()).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.null",
				scheduleBeingProcessed + " 1", "end_date");

		assertUserError(resultActions, status().isBadRequest(), errorMessage);
	}

	@Test
	@Transactional
	public void testCreateScheduleWithoutStartTimeInSpecificDateSchedules_Failure_08() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		schedules.getSpecific_date().get(0).setStartTime(null);

		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put(getCreateSchedulePath()).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.null",
				scheduleBeingProcessed + " 1", "start_time");

		assertUserError(resultActions, status().isBadRequest(), errorMessage);
	}

	@Test
	@Transactional
	public void testCreateScheduleWithoutEndTimeInSpecificDateSchedules_Failure_09() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		schedules.getSpecific_date().get(0).setEndTime(null);

		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put(getCreateSchedulePath()).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.null",
				scheduleBeingProcessed + " 1", "end_time");

		assertUserError(resultActions, status().isBadRequest(), errorMessage);

		errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.null",
				scheduleBeingProcessed + " 1", "end_time");
	}

	@Test
	@Transactional
	public void testCreateScheduleWithNullValueInstanceMaxCount_Failure_10() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		schedules.getSpecific_date().get(0).setInstanceMaxCount(null);

		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put(getCreateSchedulePath()).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.invalid",
				scheduleBeingProcessed + " 1", "instance_max_count");

		assertUserError(resultActions, status().isBadRequest(), errorMessage);
	}

	@Test
	@Transactional
	public void testCreateScheduleWithNullValueInstanceMinCount_Failure_11() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		schedules.getSpecific_date().get(0).setInstanceMinCount(null);

		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put(getCreateSchedulePath()).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.invalid",
				scheduleBeingProcessed + " 1", "instance_min_count");

		assertUserError(resultActions, status().isBadRequest(), errorMessage);
	}

	@Test
	@Transactional
	public void testCreateScheduleWithNegativeInstanceMinCount_Failure_12() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		schedules.getSpecific_date().get(0).setInstanceMinCount(-1);

		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put(getCreateSchedulePath()).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.invalid",
				scheduleBeingProcessed + " 1", "instance_min_count");

		assertUserError(resultActions, status().isBadRequest(), errorMessage);
	}

	@Test
	@Transactional
	public void testCreateScheduleWithNegativeInstanceMaxCount_Failure_13() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		schedules.getSpecific_date().get(0).setInstanceMaxCount(-1);

		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put(getCreateSchedulePath()).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.invalid",
				scheduleBeingProcessed + " 1", "instance_max_count");

		assertUserError(resultActions, status().isBadRequest(), errorMessage);
	}

	@Test
	@Transactional
	public void testCreateScheduleWithInstanceMinCountGreaterThanInstanceMaxCount_Failure_14() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		ScheduleEntity entity = schedules.getSpecific_date().get(0);
		Integer tmpInt = entity.getInstanceMaxCount();
		entity.setInstanceMaxCount(entity.getInstanceMinCount());
		entity.setInstanceMinCount(tmpInt);

		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put(getCreateSchedulePath()).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.instanceCount.invalid.min.greater",
				scheduleBeingProcessed + " 1", "instance_max_count", "instance_min_count");

		assertUserError(resultActions, status().isBadRequest(), errorMessage);
	}

	@Test
	@Transactional
	public void testCreateScheduleWithEndDateTimeBeforeStartDateTime_Failure_15() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		// Swap startTime for endTime.
		ScheduleEntity entity = schedules.getSpecific_date().get(0);
		Time tmpTime = entity.getStartTime();
		entity.setStartTime(entity.getEndTime());
		entity.setEndTime(tmpTime);

		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put(getCreateSchedulePath()).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.date.invalid.start.after.end",
				scheduleBeingProcessed + " 1", "end_date + end_time", "start_date + start_time");

		assertUserError(resultActions, status().isBadRequest(), errorMessage);
	}

	@Test
	@Transactional
	public void testCreateScheduleWithOverlappingSpecificDateSchedule_Failure_16() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 2;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		// Overlap specificDate schedules.
		ScheduleEntity firstEntity = schedules.getSpecific_date().get(0);
		ScheduleEntity secondEntity = schedules.getSpecific_date().get(1);
		secondEntity.setStartDate(firstEntity.getEndDate());
		secondEntity.setStartTime(firstEntity.getEndTime());

		String content = mapper.writeValueAsString(schedules);

		ResultActions resultActions = mockMvc
				.perform(put(getCreateSchedulePath()).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.date.overlap", scheduleBeingProcessed,
				"1", "2");
		assertUserError(resultActions, status().isBadRequest(), errorMessage);
	}

	private ResultActions getResultActionWithCreateSchedule(int noOfSpecificDateSchedulesToSetUp,
			int noOfRecurringSchedulesToSetUp) throws Exception {
		String content = TestDataSetupHelper.generateJsonSchedule(noOfSpecificDateSchedulesToSetUp,
				noOfRecurringSchedulesToSetUp);

		return mockMvc.perform(put(getCreateSchedulePath()).contentType(MediaType.APPLICATION_JSON).content(content));

	}

	private ResultActions getResultActionWithGetSchedule() throws Exception {

		return mockMvc.perform(get(getCreateSchedulePath()).accept(MediaType.APPLICATION_JSON));

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
