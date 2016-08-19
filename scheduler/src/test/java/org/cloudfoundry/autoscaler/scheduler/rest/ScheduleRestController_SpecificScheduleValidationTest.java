package org.cloudfoundry.autoscaler.scheduler.rest;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.delete;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.content;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import java.util.ArrayList;
import java.util.Collections;
import java.util.Date;
import java.util.List;

import org.cloudfoundry.autoscaler.scheduler.dao.SpecificDateScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationScalingSchedules;
import org.cloudfoundry.autoscaler.scheduler.util.DateHelper;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleTypeEnum;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
import org.hamcrest.Matchers;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.quartz.Scheduler;
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
	private SpecificDateScheduleDao specificDateScheduleDao;

	@Autowired
	private MessageBundleResourceHelper messageBundleResourceHelper;

	@Autowired
	private WebApplicationContext wac;
	private MockMvc mockMvc;

	private String appId = TestDataSetupHelper.generateAppIds(1)[0];

	private String scheduleBeingProcessed = ScheduleTypeEnum.SPECIFIC_DATE.getDescription();

	@Before
	public void beforeTest() throws Exception {
		// Clear previous schedules.
		scheduler.clear();
		mockMvc = MockMvcBuilders.webAppContextSetup(wac).build();
		removeData();
	}

	public void removeData() throws Exception {
		List<String> allAppIds = TestDataSetupHelper.getAllGeneratedAppIds();
		for (String appId : allAppIds) {
			for (SpecificDateScheduleEntity entity : specificDateScheduleDao
					.findAllSpecificDateSchedulesByAppId(appId)) {
				callDeleteSchedules(entity.getAppId());
			}
		}
	}


	@Test
	public void testCreateSchedule_without_startDateTime() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedules(noOfSpecificDateSchedulesToSetUp, 0);

		schedules.getSpecific_date().get(0).setStartDateTime(null);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "start_date_time");

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	public void testCreateSchedule_without_endDateTime() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedules(noOfSpecificDateSchedulesToSetUp, 0);

		schedules.getSpecific_date().get(0).setEndDateTime(null);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "end_date_time");

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	public void testCreateSchedule_without_instanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedules(noOfSpecificDateSchedulesToSetUp, 0);

		schedules.getSpecific_date().get(0).setInstanceMaxCount(null);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "instance_max_count");

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	public void testCreateSchedule_without_instanceMinCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedules(noOfSpecificDateSchedulesToSetUp, 0);

		schedules.getSpecific_date().get(0).setInstanceMinCount(null);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "instance_min_count");

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	public void testCreateSchedule_negative_instanceMinCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedules(noOfSpecificDateSchedulesToSetUp, 0);
		Integer instanceMinCount = -1;
		schedules.getSpecific_date().get(0).setInstanceMinCount(instanceMinCount);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.invalid",
				scheduleBeingProcessed + " 0", "instance_min_count", instanceMinCount);

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	public void testCreateSchedule_negative_instanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedules(noOfSpecificDateSchedulesToSetUp, 0);
		Integer instanceMaxCount = -1;
		schedules.getSpecific_date().get(0).setInstanceMaxCount(instanceMaxCount);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.invalid",
				scheduleBeingProcessed + " 0", "instance_max_count", instanceMaxCount);

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	public void testCreateSchedule_instanceMinCount_greater_than_instanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedules(noOfSpecificDateSchedulesToSetUp, 0);

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
	public void testCreateSchedule_startDateTime_after_endDateTime() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedules(noOfSpecificDateSchedulesToSetUp, 0);

		// Swap startDateTime and endDateTime.
		SpecificDateScheduleEntity entity = schedules.getSpecific_date().get(0);
		Date endDateTime = entity.getStartDateTime();
		Date startDateTime = entity.getEndDateTime();
		entity.setStartDateTime(startDateTime);
		entity.setEndDateTime(endDateTime);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.date.invalid.start.after.end",
				scheduleBeingProcessed + " 0", "end_date_time",
				DateHelper.convertDateTimeToString(entity.getEndDateTime()), "start_date_time",
				DateHelper.convertDateTimeToString(entity.getStartDateTime()));

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	public void testCreateSchedule_currentDateTime_after_startDateTime() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedules(noOfSpecificDateSchedulesToSetUp, 0);

		// Swap startTime for endTime.
		SpecificDateScheduleEntity entity = schedules.getSpecific_date().get(0);
		Date oldDate = new Date(0);
		entity.setStartDateTime(oldDate);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.date.invalid.current.after",
				scheduleBeingProcessed + " 0", "start_date_time",
				DateHelper.convertDateTimeToString(entity.getStartDateTime()));

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	public void testCreateSchedule_currentDateTime_after_endDate() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedules(noOfSpecificDateSchedulesToSetUp, 0);

		// Swap startTime for endTime.
		SpecificDateScheduleEntity entity = schedules.getSpecific_date().get(0);
		Date oldDate = new Date(0);
		entity.setEndDateTime(oldDate);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.date.invalid.current.after",
				scheduleBeingProcessed + " 0", "end_date_time",
				DateHelper.convertDateTimeToString(entity.getEndDateTime()));

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	public void testCreateSchedule_overlapping_date_time() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 2;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedules(noOfSpecificDateSchedulesToSetUp, 0);

		// Overlap specificDate schedules.
		SpecificDateScheduleEntity firstEntity = schedules.getSpecific_date().get(0);
		SpecificDateScheduleEntity secondEntity = schedules.getSpecific_date().get(1);
		secondEntity.setStartDateTime(firstEntity.getEndDateTime());

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.date.overlap",
				scheduleBeingProcessed + " 0", "end_date_time", scheduleBeingProcessed + " 1", "start_date_time");

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	public void testCreateSchedule_overlapping_multipleSchedules() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 4;
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedules(noOfSpecificDateSchedulesToSetUp, 0);

		// Overlap specificDate schedules.
		// Schedule 1 end date time and Schedule 2 start date time are overlapping.
		// Schedules 3 and 4 overlap with start date time.
		SpecificDateScheduleEntity firstEntity = schedules.getSpecific_date().get(0);
		SpecificDateScheduleEntity secondEntity = schedules.getSpecific_date().get(1);
		secondEntity.setStartDateTime(firstEntity.getEndDateTime());

		SpecificDateScheduleEntity thirdEntity = schedules.getSpecific_date().get(2);
		SpecificDateScheduleEntity forthEntity = schedules.getSpecific_date().get(3);
		forthEntity.setStartDateTime(thirdEntity.getStartDateTime());

		String content = mapper.writeValueAsString(schedules);

		List<String> messages = new ArrayList<>();
		messages.add(messageBundleResourceHelper.lookupMessage("schedule.date.overlap", scheduleBeingProcessed + " 0",
				"end_date_time", scheduleBeingProcessed + " 1", "start_date_time"));
		messages.add(messageBundleResourceHelper.lookupMessage("schedule.date.overlap", scheduleBeingProcessed + " 2",
				"start_date_time", scheduleBeingProcessed + " 3", "start_date_time"));

		assertErrorMessage(appId, content, messages.toArray(new String[0]));
	}

	@Test
	public void testCreateSchedule_without_specificDateSchedules() throws Exception {
		// No schedules - null case
		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedules(1, 0);

		schedules.setSpecific_date(null);

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.invalid.noSchedules",
				"app_id=" + appId);

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	public void testCreateSchedule_empty_specificDateSchedules() throws Exception {
		// No schedules - Empty case
		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedules(1, 0);

		schedules.setSpecific_date(Collections.emptyList());

		String content = mapper.writeValueAsString(schedules);

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.invalid.noSchedules",
				"app_id=" + appId);

		assertErrorMessage(appId, content, errorMessage);
	}

	@Test
	public void testCreateSchedule_without_startEndDateTime_instanceMaxMinCount() throws Exception {
		// schedules - no parameters.
		ObjectMapper mapper = new ObjectMapper();
		ApplicationScalingSchedules schedules = TestDataSetupHelper
				.generateSchedules(1, 0);

		SpecificDateScheduleEntity entity = schedules.getSpecific_date().get(0);
		entity.setInstanceMinCount(null);
		entity.setInstanceMaxCount(null);
		entity.setStartDateTime(null);
		entity.setEndDateTime(null);

		String content = mapper.writeValueAsString(schedules);

		List<String> messages = new ArrayList<>();

		messages.add(messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "start_date_time"));
		messages.add(messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "end_date_time"));
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

	private ResultActions callDeleteSchedules(String appId) throws Exception {

		return mockMvc.perform(delete(getCreateSchedulePath(appId)).accept(MediaType.APPLICATION_JSON));

	}

}
