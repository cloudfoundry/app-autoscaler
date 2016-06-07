package org.cloudfoundry.autoscaler.scheduler.rest;

import static org.hamcrest.Matchers.hasSize;
import static org.hamcrest.Matchers.is;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import org.apache.commons.logging.Log;
import org.apache.commons.logging.LogFactory;
import org.cloudfoundry.autoscaler.scheduler.util.DataSetupHelper;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.quartz.Scheduler;
import org.quartz.SchedulerException;
import org.quartz.impl.StdSchedulerFactory;
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

import com.fasterxml.jackson.core.JsonProcessingException;

/**
 * @author Fujitsu
 *
 */
@RunWith(SpringRunner.class)
@SpringBootTest(webEnvironment = WebEnvironment.RANDOM_PORT)

@DirtiesContext(classMode = ClassMode.BEFORE_EACH_TEST_METHOD)
public class ScheduleRestApiTest {

	@Autowired
	private WebApplicationContext wac;
	private MockMvc mockMvc;
	private Log logger = LogFactory.getLog(this.getClass());

	@Before
	public void beforeTest() throws SchedulerException {
		// Clear previous schedules.
		Scheduler scheduler = StdSchedulerFactory.getDefaultScheduler();
		scheduler.clear();

		mockMvc = MockMvcBuilders.webAppContextSetup(wac).build();

	}

	@Test
	public void testCreateSchedule_01() throws Exception {
		logger.info("Executing Test Create Schedule to create one schedule ...");

		logger.info("======= Create one simple schedule =======");
		int noOfSimpleSchedulesToSetUp = 1;
		assertCreateScheduleRestApi(noOfSimpleSchedulesToSetUp, 0);

		logger.info("======= Test Completed =======");
	}

	@Test
	public void testCreateSchedule_02() throws Exception {
		logger.info("Executing Test Create Schedule to create two schedules ...");

		logger.info("======= Create two simple schedules =======");
		int noOfSimpleSchedulesToSetUp = 2;
		assertCreateScheduleRestApi(noOfSimpleSchedulesToSetUp, 0);

		logger.info("======= Test Completed =======");
	}

	private void assertCreateScheduleRestApi(int noOfSimpleSchedulesToSetUp, int noOfCronSchedulesToSetUp)
			throws JsonProcessingException, Exception {
		String content = DataSetupHelper.generateJsonSchedule(noOfSimpleSchedulesToSetUp, noOfCronSchedulesToSetUp);

		ResultActions resultActions = mockMvc
				.perform(put("/v2/schedules").contentType(MediaType.APPLICATION_JSON).content(content));

		logger.info(
				"======= Check the application has " + noOfSimpleSchedulesToSetUp + " specific schedule(s) =======");
		// Checks application id exists and there is one specific schedule
		// created.
		resultActions.andExpect(status().isCreated()).andExpect(jsonPath("$.app_id", is(DataSetupHelper.getAppId())))
				.andExpect(jsonPath("$.specific_date", hasSize(noOfSimpleSchedulesToSetUp)));
	}

}
