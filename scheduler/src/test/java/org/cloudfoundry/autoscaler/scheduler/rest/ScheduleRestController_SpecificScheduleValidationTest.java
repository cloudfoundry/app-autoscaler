package org.cloudfoundry.autoscaler.scheduler.rest;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.content;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import java.sql.Time;
import java.util.ArrayList;
import java.util.Collections;
import java.util.Date;
import java.util.List;

import javax.transaction.Transactional;

import org.cloudfoundry.autoscaler.scheduler.dao.ScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationScalingSchedules;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleTypeEnum;
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
public class ScheduleRestController_SpecificScheduleValidationTest {

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

	private String scheduleBeingProcessed = ScheduleTypeEnum.SPECIFIC_DATE.getDescription();

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
			for (ScheduleEntity entity : scheduleDao.findAllSchedulesByAppId(appId)) {
				scheduleDao.delete(entity);
			}
		}
	}


	@Test
	@Transactional
	public void testCreateSchedule_without_startDate() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		schedules.getSpecific_date().get(0).setStartDate(null);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "start_date");

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_without_endDate() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		schedules.getSpecific_date().get(0).setEndDate(null);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "end_date");

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_without_startTime() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		schedules.getSpecific_date().get(0).setStartTime(null);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "start_time");

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_without_endTime() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		schedules.getSpecific_date().get(0).setEndTime(null);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "end_time");

		assertErrorMessage(appId, content, errorMessage);

	}

	@Test
	@Transactional
	public void testCreateSchedule_without_instanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		schedules.getSpecific_date().get(0).setInstanceMaxCount(null);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "instance_max_count");

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_without_instanceMinCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		schedules.getSpecific_date().get(0).setInstanceMinCount(null);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "instance_min_count");

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_negative_instanceMinCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);
		Integer instanceMinCount = -1;
		schedules.getSpecific_date().get(0).setInstanceMinCount(instanceMinCount);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.invalid",
				scheduleBeingProcessed + " 0", "instance_min_count", instanceMinCount);

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_negative_instanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);
		Integer instanceMaxCount = -1;
		schedules.getSpecific_date().get(0).setInstanceMaxCount(instanceMaxCount);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.invalid",
				scheduleBeingProcessed + " 0", "instance_max_count", instanceMaxCount);

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_instanceMinCount_greater_than_instanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		ScheduleEntity entity = schedules.getSpecific_date().get(0);
		Integer instanceMinCount = 5;
		Integer instanceMaxCount = 4;
		entity.setInstanceMaxCount(instanceMaxCount);
		entity.setInstanceMinCount(instanceMinCount);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.instanceCount.invalid.min.greater",
				scheduleBeingProcessed + " 0", "instance_max_count", instanceMaxCount, "instance_min_count",
				instanceMinCount);

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_startDateTime_after_endDateTime() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		// Swap startTime for endTime.
		ScheduleEntity entity = schedules.getSpecific_date().get(0);
		Time endTime = entity.getStartTime();
		Time startTime = entity.getEndTime();
		entity.setStartTime(startTime);
		entity.setEndTime(endTime);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.date.invalid.start.after.end",
				scheduleBeingProcessed + " 0", "end_date end_time", entity.getEndDate() + " " + endTime,
				"start_date start_time", entity.getStartDate() + " " + startTime);

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_currentDateTime_after_startDateTime() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		// Swap startTime for endTime.
		ScheduleEntity entity = schedules.getSpecific_date().get(0);
		Date oldDate = new Date(0);
		entity.setStartDate(oldDate);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.date.invalid.current.after",
				scheduleBeingProcessed + " 0", "start_date start_time", entity.getStartDate(), entity.getStartTime());

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_currentDateTime_after_endDate() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		// Swap startTime for endTime.
		ScheduleEntity entity = schedules.getSpecific_date().get(0);
		Date oldDate = new Date(0);
		entity.setEndDate(oldDate);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.date.invalid.current.after",
				scheduleBeingProcessed + " 0", "end_date end_time", entity.getEndDate(), entity.getEndTime());

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_overlapping_date_time() throws Exception {

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
				"0", "end_date end_time", "1", "start_date start_time");

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_overlapping_multipleSchedules() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 4;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, noOfSpecificDateSchedulesToSetUp);

		// Overlap specificDate schedules.
		// Schedule 1 end date, end time and Schedule 2 start date, start time are overlapping.
		// Schedules 3 and 4 is overlap with start date and start time.
		ScheduleEntity firstEntity = schedules.getSpecific_date().get(0);
		ScheduleEntity secondEntity = schedules.getSpecific_date().get(1);
		secondEntity.setStartDate(firstEntity.getEndDate());
		secondEntity.setStartTime(firstEntity.getEndTime());

		ScheduleEntity thirdEntity = schedules.getSpecific_date().get(2);
		ScheduleEntity forthEntity = schedules.getSpecific_date().get(3);
		forthEntity.setStartDate(thirdEntity.getStartDate());
		forthEntity.setStartTime(thirdEntity.getStartTime());

		String content = mapper.writeValueAsString(schedules);

		List<String> messages = new ArrayList<>();
		messages.add(messageBundleResourceHelper.lookupMessage("schedule.date.overlap", scheduleBeingProcessed, "0",
				"end_date end_time", "1", "start_date start_time"));
		messages.add(messageBundleResourceHelper.lookupMessage("schedule.date.overlap", scheduleBeingProcessed, "2",
				"start_date start_time", "3", "start_date start_time"));

		assertErrorMessage(appId, content, messages.toArray(new String[0]));
	}

	@Test
	@Transactional
	public void testCreateSchedule_without_specificDateSchedules() throws Exception {
		// No schedules - null case
		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, 1);

		schedules.setSpecific_date(null);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.invalid.noSchedules",
				"app_id=" + appId);

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_empty_specificDateSchedules() throws Exception {
		// No schedules - Empty case
		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, 1);

		schedules.setSpecific_date(Collections.<ScheduleEntity> emptyList());

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.invalid.noSchedules",
				"app_id=" + appId);

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	@Transactional
	public void testCreateSchedule_without_startEndDateTime_instanceMaxMinCount() throws Exception {
		// schedules - no parameters.
		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSpecificDateSchedulesForScheduleController(appId, 1);

		ScheduleEntity entity = schedules.getSpecific_date().get(0);
		entity.setInstanceMinCount(null);
		entity.setInstanceMaxCount(null);
		entity.setStartDate(null);
		entity.setStartTime(null);
		entity.setEndDate(null);
		entity.setEndTime(null);

		String content = mapper.writeValueAsString(schedules);

		List<String> messages = new ArrayList<>();

		messages.add(messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "start_date"));
		messages.add(messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "end_date"));
		messages.add(messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "start_time"));
		messages.add(messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "end_time"));
		messages.add(messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "instance_max_count"));
		messages.add(messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "instance_min_count"));

		assertErrorMessage(appId, content, messages.toArray(new String[0]));
	}

	private void assertErrorMessage(String appId, String inputContent, String... expectedErrorMessages)
			throws Exception {
		ResultActions resultActions = mockMvc.perform(
				put(getCreateSchedulePath(appId)).contentType(MediaType.APPLICATION_JSON).content(inputContent));

		resultActions.andExpect(status().isBadRequest());
		resultActions.andExpect(content().contentTypeCompatibleWith(MediaType.APPLICATION_JSON));
		resultActions.andExpect(jsonPath("$").isArray());
		resultActions.andExpect(jsonPath("$").value(Matchers.containsInAnyOrder(expectedErrorMessages)));
	}

	private String getCreateSchedulePath(String appId) {
		return String.format("/v2/schedules/%s", appId);
	}

}
