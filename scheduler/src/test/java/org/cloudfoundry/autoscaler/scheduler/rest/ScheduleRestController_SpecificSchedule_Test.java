package org.cloudfoundry.autoscaler.scheduler.rest;

import static org.junit.Assert.assertEquals;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.content;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.header;
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

	private String[] multiAppIds = TestDataSetupHelper.getAppIds();
	private String[] singleAppId = { TestDataSetupHelper.getAppId_1() };
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
		for (String appId : multiAppIds) {
			for (ScheduleEntity entity : scheduleDao.findAllSchedulesByAppId(appId)) {
				scheduleDao.delete(entity);
			}
		}
	}

	@Test
	@Transactional
	public void testGetAllSchedule_with_no_schedules() throws Exception {
		ResultActions resultActions = callGetAllSchedulesByAppId(appId);
		assertSpecificSchedulesFoundEquals(0, resultActions);
	}

	@Test
	@Transactional
	public void testCreateAndGetSchedules() throws Exception {
		assertCreateAndGetSchedules(singleAppId, 1);
		assertCreateAndGetSchedules(singleAppId, 5);

		assertCreateAndGetSchedules(multiAppIds, 1);
		assertCreateAndGetSchedules(multiAppIds, 5);
	}

	@Test
	@Transactional
	public void testCreateSchedule_withoutStartDate_Failure() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		schedules.getSpecific_date().get(0).setStartDate(null);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.null",
				scheduleBeingProcessed + " 1", "start_date");

		assertErrorMessage(content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_withoutEndDate_Failure() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		schedules.getSpecific_date().get(0).setEndDate(null);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.null",
				scheduleBeingProcessed + " 1", "end_date");

		assertErrorMessage(content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_withoutStartTimeInSpecificDateSchedules_Failure() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		schedules.getSpecific_date().get(0).setStartTime(null);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.null",
				scheduleBeingProcessed + " 1", "start_time");

		assertErrorMessage(content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_withoutEndTimeInSpecificDateSchedules_Failure() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		schedules.getSpecific_date().get(0).setEndTime(null);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.null",
				scheduleBeingProcessed + " 1", "end_time");

		assertErrorMessage(content, errorMessage);

	}

	@Test
	@Transactional
	public void testCreateSchedule_withNullValueInstanceMaxCount_Failure() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		schedules.getSpecific_date().get(0).setInstanceMaxCount(null);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.invalid",
				scheduleBeingProcessed + " 1", "instance_max_count");

		assertErrorMessage(content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_withNullValueInstanceMinCount_Failure() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		schedules.getSpecific_date().get(0).setInstanceMinCount(null);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.invalid",
				scheduleBeingProcessed + " 1", "instance_min_count");

		assertErrorMessage(content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_withNegativeInstanceMinCount_Failure() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		schedules.getSpecific_date().get(0).setInstanceMinCount(-1);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.invalid",
				scheduleBeingProcessed + " 1", "instance_min_count");

		assertErrorMessage(content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_withNegativeInstanceMaxCount_Failure() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		schedules.getSpecific_date().get(0).setInstanceMaxCount(-1);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.invalid",
				scheduleBeingProcessed + " 1", "instance_max_count");

		assertErrorMessage(content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_withInstanceMinCountGreaterThanInstanceMaxCount_Failure() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		ScheduleEntity entity = schedules.getSpecific_date().get(0);
		Integer tmpInt = entity.getInstanceMaxCount();
		entity.setInstanceMaxCount(entity.getInstanceMinCount());
		entity.setInstanceMinCount(tmpInt);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.instanceCount.invalid.min.greater",
				scheduleBeingProcessed + " 1", "instance_max_count", "instance_min_count");

		assertErrorMessage(content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_withEndDateTimeBeforeStartDateTime_Failure() throws Exception {

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

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.date.invalid.start.after.end",
				scheduleBeingProcessed + " 1", "end_date + end_time", "start_date + start_time");

		assertErrorMessage(content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_withOverlappingSpecificDateSchedule_Failure() throws Exception {

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

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.date.overlap", scheduleBeingProcessed,
				"1", "2");

		assertErrorMessage(content, errorMessage);
	}

	private void assertErrorMessage(String inputContent, String expectedErrorMessage) throws Exception {
		ResultActions resultActions = mockMvc.perform(
				put(getCreateSchedulePath(appId)).contentType(MediaType.APPLICATION_JSON).content(inputContent));

		resultActions = mockMvc.perform(
				put(getCreateSchedulePath(appId)).contentType(MediaType.APPLICATION_JSON).content(inputContent));

		assertUserError(resultActions, status().isBadRequest(), expectedErrorMessage);
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

	private void assertUserError(ResultActions resultActions, ResultMatcher statusCode, String message)
			throws Exception {
		resultActions.andExpect(statusCode);
		resultActions.andExpect(content().contentTypeCompatibleWith(MediaType.APPLICATION_JSON));
		resultActions.andExpect(jsonPath("$").isArray());
		resultActions.andExpect(jsonPath("$[0]").value(Matchers.containsString(message)));
	}
}
