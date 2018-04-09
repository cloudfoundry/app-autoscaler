package org.cloudfoundry.autoscaler.scheduler.rest;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.content;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

import org.cloudfoundry.autoscaler.scheduler.dao.RecurringScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.dao.SpecificDateScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationSchedules;
import org.cloudfoundry.autoscaler.scheduler.util.DateHelper;
import org.cloudfoundry.autoscaler.scheduler.util.ScheduleTypeEnum;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataDbUtil;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataSetupHelper;
import org.cloudfoundry.autoscaler.scheduler.util.error.MessageBundleResourceHelper;
import org.hamcrest.Matchers;
import org.junit.Before;
import org.junit.Test;
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
public class ScheduleRestController_SpecificScheduleValidationTest {

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

	private String appId = TestDataSetupHelper.generateAppIds(1)[0];
	String guid = TestDataSetupHelper.generateGuid();

	private String scheduleBeingProcessed = ScheduleTypeEnum.SPECIFIC_DATE.getDescription();

	@Before
	public void beforeTest() throws Exception {
		testDataDbUtil.cleanupData();
		Mockito.reset(scheduler);

		mockMvc = MockMvcBuilders.webAppContextSetup(wac).build();
	}

	@Test
	public void testCreateSchedule_without_startDateTime() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper
				.generateApplicationPolicy(noOfSpecificDateSchedulesToSetUp, 0);

		applicationPolicy.getSchedules().getSpecificDate().get(0).setStartDateTime(null);

		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "start_date_time");
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_without_endDateTime() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper
				.generateApplicationPolicy(noOfSpecificDateSchedulesToSetUp, 0);

		applicationPolicy.getSchedules().getSpecificDate().get(0).setEndDateTime(null);

		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "end_date_time");
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_without_instanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper
				.generateApplicationPolicy(noOfSpecificDateSchedulesToSetUp, 0);

		applicationPolicy.getSchedules().getSpecificDate().get(0).setInstanceMaxCount(null);

		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "instance_max_count");
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_without_instanceMinCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper
				.generateApplicationPolicy(noOfSpecificDateSchedulesToSetUp, 0);

		applicationPolicy.getSchedules().getSpecificDate().get(0).setInstanceMinCount(null);

		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "instance_min_count");
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_negative_instanceMinCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper
				.generateApplicationPolicy(noOfSpecificDateSchedulesToSetUp, 0);
		Integer instanceMinCount = -1;
		applicationPolicy.getSchedules().getSpecificDate().get(0).setInstanceMinCount(instanceMinCount);

		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.invalid",
				scheduleBeingProcessed + " 0", "instance_min_count", instanceMinCount);
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_negative_instanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper
				.generateApplicationPolicy(noOfSpecificDateSchedulesToSetUp, 0);
		Integer instanceMaxCount = -1;
		applicationPolicy.getSchedules().getSpecificDate().get(0).setInstanceMaxCount(instanceMaxCount);

		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.invalid",
				scheduleBeingProcessed + " 0", "instance_max_count", instanceMaxCount);
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_instanceMinCount_greater_than_instanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper
				.generateApplicationPolicy(noOfSpecificDateSchedulesToSetUp, 0);

		ScheduleEntity entity = applicationPolicy.getSchedules().getSpecificDate().get(0);
		Integer instanceMinCount = 5;
		Integer instanceMaxCount = 4;
		entity.setInstanceMaxCount(instanceMaxCount);
		entity.setInstanceMinCount(instanceMinCount);

		String content = mapper.writeValueAsString(applicationPolicy);

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
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper
				.generateApplicationPolicy(noOfSpecificDateSchedulesToSetUp, 0);

		applicationPolicy.getSchedules().getSpecificDate().get(0).setInitialMinInstanceCount(5);

		String content = mapper.writeValueAsString(applicationPolicy);
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		assertResponseStatusEquals(resultActions, status().isOk());
	}

	@Test
	public void testCreateSchedule_negative_initialMinInstanceCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper
				.generateApplicationPolicy(noOfSpecificDateSchedulesToSetUp, 0);
		Integer initialMinInstanceCount = -1;
		applicationPolicy.getSchedules().getSpecificDate().get(0).setInitialMinInstanceCount(initialMinInstanceCount);

		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.data.value.invalid",
				scheduleBeingProcessed + " 0", "initial_min_instance_count", initialMinInstanceCount);
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_startDateTime_after_endDateTime() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper
				.generateApplicationPolicy(noOfSpecificDateSchedulesToSetUp, 0);

		// Swap startDateTime and endDateTime.
		SpecificDateScheduleEntity entity = applicationPolicy.getSchedules().getSpecificDate().get(0);
		LocalDateTime endDateTime = entity.getStartDateTime();
		LocalDateTime startDateTime = entity.getEndDateTime();
		entity.setStartDateTime(startDateTime);
		entity.setEndDateTime(endDateTime);

		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.date.invalid.start.after.end",
				scheduleBeingProcessed + " 0", "end_date_time",
				DateHelper.convertLocalDateTimeToString(entity.getEndDateTime()), "start_date_time",
				DateHelper.convertLocalDateTimeToString(entity.getStartDateTime()));
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_currentDateTime_after_startDateTime() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper
				.generateApplicationPolicy(noOfSpecificDateSchedulesToSetUp, 0);

		// Swap startTime for endTime.
		SpecificDateScheduleEntity entity = applicationPolicy.getSchedules().getSpecificDate().get(0);
		LocalDateTime oldDate = LocalDateTime.now().minusDays(1);
		entity.setStartDateTime(oldDate);

		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.date.invalid.current.after",
				scheduleBeingProcessed + " 0", "start_date_time",
				DateHelper.convertLocalDateTimeToString(entity.getStartDateTime()));
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_currentDateTime_after_endDate() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 1;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper
				.generateApplicationPolicy(noOfSpecificDateSchedulesToSetUp, 0);

		// Swap startTime for endTime.
		SpecificDateScheduleEntity entity = applicationPolicy.getSchedules().getSpecificDate().get(0);
		LocalDateTime oldDate = LocalDateTime.now().minusDays(1);
		entity.setEndDateTime(oldDate);

		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.date.invalid.current.after",
				scheduleBeingProcessed + " 0", "end_date_time",
				DateHelper.convertLocalDateTimeToString(entity.getEndDateTime()));
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_overlapping_date_time() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 2;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper
				.generateApplicationPolicy(noOfSpecificDateSchedulesToSetUp, 0);

		// Overlap specificDate schedules.
		SpecificDateScheduleEntity firstEntity = applicationPolicy.getSchedules().getSpecificDate().get(0);
		SpecificDateScheduleEntity secondEntity = applicationPolicy.getSchedules().getSpecificDate().get(1);
		secondEntity.setStartDateTime(firstEntity.getEndDateTime());

		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		String errorMessage = messageBundleResourceHelper.lookupMessage("schedule.date.overlap",
				scheduleBeingProcessed + " 0", "end_date_time", scheduleBeingProcessed + " 1", "start_date_time");
		assertErrorMessage(resultActions, errorMessage);
	}

	@Test
	public void testCreateSchedule_overlapping_multipleSchedules() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		int noOfSpecificDateSchedulesToSetUp = 4;
		ApplicationSchedules applicationPolicy = TestDataSetupHelper
				.generateApplicationPolicy(noOfSpecificDateSchedulesToSetUp, 0);

		// Overlap specificDate schedules.
		// Schedule 1 end date time and Schedule 2 start date time are overlapping.
		// Schedules 3 and 4 overlap with start date time.
		SpecificDateScheduleEntity firstEntity = applicationPolicy.getSchedules().getSpecificDate().get(0);
		SpecificDateScheduleEntity secondEntity = applicationPolicy.getSchedules().getSpecificDate().get(1);
		secondEntity.setStartDateTime(firstEntity.getEndDateTime());

		SpecificDateScheduleEntity thirdEntity = applicationPolicy.getSchedules().getSpecificDate().get(2);
		SpecificDateScheduleEntity forthEntity = applicationPolicy.getSchedules().getSpecificDate().get(3);
		forthEntity.setStartDateTime(thirdEntity.getStartDateTime());

		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		List<String> messages = new ArrayList<>();
		messages.add(messageBundleResourceHelper.lookupMessage("schedule.date.overlap", scheduleBeingProcessed + " 0",
				"end_date_time", scheduleBeingProcessed + " 1", "start_date_time"));
		messages.add(messageBundleResourceHelper.lookupMessage("schedule.date.overlap", scheduleBeingProcessed + " 2",
				"start_date_time", scheduleBeingProcessed + " 3", "start_date_time"));

		assertErrorMessage(resultActions, messages.toArray(new String[0]));
	}

	@Test
	public void testCreateSchedule_without_specificDateSchedules() throws Exception {
		// No schedules - null case
		ObjectMapper mapper = new ObjectMapper();
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);

		applicationPolicy.getSchedules().setSpecificDate(null);

		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		assertResponseStatusEquals(resultActions, status().isOk());
	}

	@Test
	public void testCreateSchedule_empty_specificDateSchedules() throws Exception {
		// No schedules - Empty case
		ObjectMapper mapper = new ObjectMapper();
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);

		applicationPolicy.getSchedules().setSpecificDate(Collections.emptyList());

		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		assertResponseStatusEquals(resultActions, status().isOk());
	}

	@Test
	public void testCreateSchedule_without_startEndDateTime_instanceMaxMinCount() throws Exception {
		// schedules - no parameters.
		ObjectMapper mapper = new ObjectMapper();
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);

		SpecificDateScheduleEntity entity = applicationPolicy.getSchedules().getSpecificDate().get(0);
		entity.setInstanceMinCount(null);
		entity.setInstanceMaxCount(null);
		entity.setStartDateTime(null);
		entity.setEndDateTime(null);

		String content = mapper.writeValueAsString(applicationPolicy);
		ResultActions resultActions = mockMvc
				.perform(put(TestDataSetupHelper.getSchedulerPath(appId)).param("guid", guid).contentType(MediaType.APPLICATION_JSON).content(content));

		List<String> messages = new ArrayList<>();
		messages.add(messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "start_date_time"));
		messages.add(messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "end_date_time"));
		messages.add(messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "instance_max_count"));
		messages.add(messageBundleResourceHelper.lookupMessage("schedule.data.value.not.specified",
				scheduleBeingProcessed + " 0", "instance_min_count"));

		assertErrorMessage(resultActions, messages.toArray(new String[0]));
	}

	private void assertErrorMessage(ResultActions resultActions, String... expectedErrorMessages) throws Exception {
		resultActions.andExpect(jsonPath("$").value(Matchers.containsInAnyOrder(expectedErrorMessages)));
		resultActions.andExpect(jsonPath("$").isArray());
		resultActions.andExpect(content().contentTypeCompatibleWith(MediaType.APPLICATION_JSON));
		resultActions.andExpect(status().isBadRequest());
	}

	private void assertResponseStatusEquals(ResultActions resultActions, ResultMatcher expectedStatus)
			throws Exception {
		resultActions.andExpect(expectedStatus);
	}
}
