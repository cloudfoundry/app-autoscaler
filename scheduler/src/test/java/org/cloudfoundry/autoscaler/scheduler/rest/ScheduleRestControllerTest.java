package org.cloudfoundry.autoscaler.scheduler.rest;

import static org.junit.Assert.assertEquals;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.delete;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.content;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.header;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.text.DateFormat;
import java.util.ArrayList;
import java.util.List;

import org.cloudfoundry.autoscaler.scheduler.dao.SpecificDateScheduleDao;
import org.cloudfoundry.autoscaler.scheduler.entity.RecurringScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.ScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.entity.SpecificDateScheduleEntity;
import org.cloudfoundry.autoscaler.scheduler.rest.model.ApplicationSchedules;
import org.cloudfoundry.autoscaler.scheduler.util.TestConfiguration;
import org.cloudfoundry.autoscaler.scheduler.util.TestDataCleanupHelper;
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
@DirtiesContext(classMode = ClassMode.BEFORE_CLASS)
public class ScheduleRestControllerTest extends TestConfiguration {

	@Autowired
	private Scheduler scheduler;

	@Autowired
	private SpecificDateScheduleDao specificDateScheduleDao;

	@Autowired
	private MessageBundleResourceHelper messageBundleResourceHelper;

	@Autowired
	private TestDataCleanupHelper testDataCleanupHelper;

	@Autowired
	private WebApplicationContext wac;
	private MockMvc mockMvc;

	private String appId = TestDataSetupHelper.generateAppIds(1)[0];

	@Before
	public void before() throws Exception {
		// Clean up data.
		testDataCleanupHelper.cleanupData(scheduler);

		mockMvc = MockMvcBuilders.webAppContextSetup(wac).build();
	}

	@Test
	public void testGetAllSchedule_with_no_schedules() throws Exception {
		ResultActions resultActions = callGetAllSchedulesByAppId(appId);

		assertNoSchedulesFound(resultActions);
	}
	
	@Test
	public void testCreateAndGetSchedules_from_jsonFile() throws Exception{
		String policyJsonStr = getPolicyJsonContent();
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		ResultActions resultActions = mockMvc.perform(put(getCreateSchedulePath(appId))
				.contentType(MediaType.APPLICATION_JSON).accept(MediaType.APPLICATION_JSON).content(policyJsonStr));
		assertCreateScheduleAPI(resultActions, status().isOk());

		resultActions = callGetAllSchedulesByAppId(appId);
		assertSchedulesFoundEquals(2, 4, appId, resultActions);
	}

	@Test
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
	public void testCreateSchedule_already_existing_schedule_for_appId() throws Exception {
		String appId = TestDataSetupHelper.generateAppIds(1)[0];
		// Create one specific date schedule and two recurring schedules for the application.
		callCreateSchedules(appId, 1, 2);

		// Create two specific date schedules and one recurring schedule for the same application.
		ResultActions resultActions = callCreateSchedules(appId, 2, 1);
		assertCreateScheduleAPI(resultActions, status().isNoContent());

		resultActions = callGetAllSchedulesByAppId(appId);
		assertSchedulesFoundEquals(2, 1, appId, resultActions);

	}

	@Test
	public void testCreateSchedule_without_appId() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);
		String content = mapper.writeValueAsString(applicationPolicy);

		ResultActions resultActions = mockMvc
				.perform(put("/v2/schedules").contentType(MediaType.APPLICATION_JSON).content(content));

		resultActions.andExpect(status().isNotFound());

	}

	@Test
	public void testCreateSchedule_without_timeZone() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);

		applicationPolicy.getSchedules().setTimeZone(null);

		String content = mapper.writeValueAsString(applicationPolicy);

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.value.not.specified.timezone",
				"timeZone");

		assertErrorMessages(appId, content, errorMessage);
	}

	@Test
	public void testCreateSchedule_empty_timeZone() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);

		applicationPolicy.getSchedules().setTimeZone("");

		String content = mapper.writeValueAsString(applicationPolicy);

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.value.not.specified.timezone",
				"timeZone");

		assertErrorMessages(appId, content, errorMessage);
	}

	@Test
	public void testCreateSchedule_invalid_timeZone() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);

		applicationPolicy.getSchedules().setTimeZone(TestDataSetupHelper.getInvalidTimezone());

		String content = mapper.writeValueAsString(applicationPolicy);

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.invalid.timezone", "timeZone");

		assertErrorMessages(appId, content, errorMessage);
	}

	@Test
	public void testCreateSchedule_without_defaultInstanceMinCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);

		applicationPolicy.setInstanceMinCount(null);

		String content = mapper.writeValueAsString(applicationPolicy);

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.default.value.not.specified",
				"instance_min_count");

		assertErrorMessages(appId, content, errorMessage);
	}

	@Test
	public void testCreateSchedule_without_defaultInstanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);

		applicationPolicy.setInstanceMaxCount(null);

		String content = mapper.writeValueAsString(applicationPolicy);

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.default.value.not.specified",
				"instance_max_count");

		assertErrorMessages(appId, content, errorMessage);
	}

	@Test
	public void testCreateSchedule_negative_defaultInstanceMinCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);
		int instanceMinCount = -1;
		applicationPolicy.setInstanceMinCount(instanceMinCount);

		String content = mapper.writeValueAsString(applicationPolicy);

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.default.value.invalid",
				"instance_min_count", instanceMinCount);

		assertErrorMessages(appId, content, errorMessage);
	}

	@Test
	public void testCreateSchedule_negative_defaultInstanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);
		int instanceMaxCount = -1;
		applicationPolicy.setInstanceMaxCount(instanceMaxCount);

		String content = mapper.writeValueAsString(applicationPolicy);

		String errorMessage = messageBundleResourceHelper.lookupMessage("data.default.value.invalid",
				"instance_max_count", instanceMaxCount);

		assertErrorMessages(appId, content, errorMessage);
	}

	@Test
	public void testCreateSchedule_defaultInstanceMinCount_greater_than_defaultInstanceMaxCount() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);

		Integer instanceMinCount = 5;
		Integer instanceMaxCount = 1;
		applicationPolicy.setInstanceMaxCount(instanceMaxCount);
		applicationPolicy.setInstanceMinCount(instanceMinCount);

		String content = mapper.writeValueAsString(applicationPolicy);

		String errorMessage = messageBundleResourceHelper.lookupMessage(
				"data.default.instanceCount.invalid.min.greater", "instance_max_count", instanceMaxCount,
				"instance_min_count", instanceMinCount);

		assertErrorMessages(appId, content, errorMessage);
	}

	@Test
	public void testCreateSchedule_without_instanceMaxAndMinCount_timeZone() throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		ApplicationSchedules applicationPolicy = TestDataSetupHelper.generateApplicationPolicy(1, 0);

		applicationPolicy.setInstanceMaxCount(null);
		applicationPolicy.setInstanceMinCount(null);
		applicationPolicy.getSchedules().setTimeZone(null);

		String content = mapper.writeValueAsString(applicationPolicy);

		List<String> messages = new ArrayList<>();
		messages.add(
				messageBundleResourceHelper.lookupMessage("data.default.value.not.specified", "instance_min_count"));
		messages.add(
				messageBundleResourceHelper.lookupMessage("data.default.value.not.specified", "instance_max_count"));
		messages.add(messageBundleResourceHelper.lookupMessage("data.value.not.specified.timezone", "timeZone"));

		assertErrorMessages(appId, content, messages.toArray(new String[0]));
	}

	@Test
	public void testCreateSchedule_multiple_error() throws Exception {
		// Should be individual each test.

		testCreateSchedule_negative_defaultInstanceMaxCount();

		testCreateSchedule_without_defaultInstanceMinCount();

		testCreateSchedule_without_defaultInstanceMaxCount();

		testCreateSchedule_defaultInstanceMinCount_greater_than_defaultInstanceMaxCount();
	}

	@Test
	public void testDeleteSchedules() throws Exception {

		// Test multiple applications each having multiple specific date schedules, no recurring schedule
		String[] multipleAppIds = TestDataSetupHelper.generateAppIds(5);
		assertDeleteSchedules(multipleAppIds, 5, 0);

		// Test multiple applications each having multiple recurring schedule, no specific date schedules
		assertDeleteSchedules(multipleAppIds, 0, 5);

		// Test multiple applications each having multiple specific date and multiple recurring schedule 
		assertDeleteSchedules(multipleAppIds, 5, 5);

	}

	@Test
	public void testDeleteSchedules_appId_without_schedules() throws Exception {
		String[] multipleAppIds = TestDataSetupHelper.generateAppIds(2);

		//  Get schedules and assert to check no schedules exist
		for (String appId : multipleAppIds) {
			ResultActions resultActions = callGetAllSchedulesByAppId(appId);
			assertNoSchedulesFound(resultActions);
		}

		for (String appId : multipleAppIds) {
			ResultActions resultActions = callDeleteSchedules(appId);
			assertNoSchedulesFound(resultActions);
		}

	}

	private String getCreateSchedulePath(String appId) {
		return String.format("/v2/schedules/%s", appId);
	}

	private ResultActions callCreateSchedules(String appId, int noOfSpecificDateSchedulesToSetUp,
			int noOfRecurringSchedulesToSetUp) throws Exception {
		String content = TestDataSetupHelper.generateJsonSchedule(appId, noOfSpecificDateSchedulesToSetUp,
				noOfRecurringSchedulesToSetUp);

		return mockMvc.perform(put(getCreateSchedulePath(appId)).contentType(MediaType.APPLICATION_JSON)
				.accept(MediaType.APPLICATION_JSON).content(content));

	}

	private ResultActions callGetAllSchedulesByAppId(String appId) throws Exception {

		return mockMvc.perform(get(getCreateSchedulePath(appId)).accept(MediaType.APPLICATION_JSON));

	}

	private ResultActions callDeleteSchedules(String appId) throws Exception {

		return mockMvc.perform(delete(getCreateSchedulePath(appId)).accept(MediaType.APPLICATION_JSON));

	}

	private void assertNoSchedulesFound(ResultActions resultActions) throws Exception {
		resultActions.andExpect(status().isNotFound());
		resultActions.andExpect(header().doesNotExist("Content-type"));
		resultActions.andExpect(content().string(Matchers.isEmptyString()));
	}

	private void assertCreateAndGetSchedules(String[] appIds, int expectedSpecificDateSchedulesTobeFound,
			int expectedRecurringScheduleTobeFound) throws Exception {

		for (String appId : appIds) {
			ResultActions resultActions = callCreateSchedules(appId, expectedSpecificDateSchedulesTobeFound,
					expectedRecurringScheduleTobeFound);
			assertCreateScheduleAPI(resultActions, status().isOk());
		}

		for (String appId : appIds) {
			ResultActions resultActions = callGetAllSchedulesByAppId(appId);
			assertSchedulesFoundEquals(expectedSpecificDateSchedulesTobeFound, expectedRecurringScheduleTobeFound,
					appId, resultActions);
		}
	}

	private void assertCreateScheduleAPI(ResultActions resultActions, ResultMatcher status) throws Exception {
		resultActions.andExpect(status);
		resultActions.andExpect(header().doesNotExist("Content-type"));
		resultActions.andExpect(content().string(Matchers.isEmptyString()));
	}

	private void assertSchedulesFoundEquals(int expectedSpecificDateSchedulesTobeFound,
			int expectedRecurringSchedulesTobeFound, String appId, ResultActions resultActions) throws Exception {

		ObjectMapper mapper = new ObjectMapper();
		mapper.setDateFormat(DateFormat.getDateInstance(DateFormat.LONG));
		ApplicationSchedules applicationPolicy = mapper.readValue(
				resultActions.andReturn().getResponse().getContentAsString(), ApplicationSchedules.class);

		resultActions.andExpect(status().isOk());
		resultActions.andExpect(content().contentTypeCompatibleWith(MediaType.APPLICATION_JSON));

		assertSpecificDateScheduleFoundEquals(expectedSpecificDateSchedulesTobeFound, appId,
				applicationPolicy.getSchedules().getSpecificDate());
		assertRecurringDateScheduleFoundEquals(expectedRecurringSchedulesTobeFound, appId,
				applicationPolicy.getSchedules().getRecurringSchedule());
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

	private void assertDeleteSchedules(String[] multipleAppIds, int specificDateSchedules, int recurringSchedules)
			throws Exception {
		for (String appId : multipleAppIds) {
			callCreateSchedules(appId, specificDateSchedules, recurringSchedules);
		}

		// Get schedules and assert to check schedules got created
		for (String appId : multipleAppIds) {
			ResultActions resultActions = callGetAllSchedulesByAppId(appId);
			assertSchedulesFoundEquals(specificDateSchedules, recurringSchedules, appId, resultActions);
		}

		for (String appId : multipleAppIds) {
			ResultActions resultActions = callDeleteSchedules(appId);
			assertSchedulesAreDeleted(resultActions);
		}

		//  Get schedules and assert to check no schedules exist
		for (String appId : multipleAppIds) {
			ResultActions resultActions = callGetAllSchedulesByAppId(appId);
			assertNoSchedulesFound(resultActions);
		}
	}

	private void assertSchedulesAreDeleted(ResultActions resultActions) throws Exception {
		resultActions.andExpect(status().isNoContent());
		resultActions.andExpect(header().doesNotExist("Content-type"));
		resultActions.andExpect(content().string(Matchers.isEmptyString()));
	}
	
	public static String getPolicyJsonContent(){
        BufferedReader br = new BufferedReader(new InputStreamReader(ApplicationSchedules.class.getResourceAsStream("/fakePolicy.json")));
        String tmp = "";
        String jsonPolicyStr = "";
        try {
            while((tmp = br.readLine()) != null){
                jsonPolicyStr += tmp;
            }
        } catch (IOException e) {
            e.printStackTrace();
        }
        jsonPolicyStr = jsonPolicyStr.replaceAll("\\s+", " ");
        return jsonPolicyStr;
    }
}
