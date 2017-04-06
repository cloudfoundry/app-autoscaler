package org.cloudfoundry.autoscaler.scheduler.util;

import static org.hamcrest.core.Is.is;
import static org.junit.Assert.assertNull;
import static org.junit.Assert.assertThat;

import java.io.IOException;
import java.util.Map;

import org.junit.After;
import org.junit.AfterClass;
import org.junit.Before;
import org.junit.BeforeClass;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.context.embedded.EmbeddedWebApplicationContext;
import org.springframework.boot.context.embedded.LocalServerPort;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.annotation.DirtiesContext;
import org.springframework.test.context.junit4.SpringRunner;

import com.ecwid.consul.v1.ConsulClient;
import com.ecwid.consul.v1.Response;
import com.ecwid.consul.v1.agent.model.Check;
import com.ecwid.consul.v1.agent.model.Service;

@RunWith(SpringRunner.class)
@SpringBootTest(webEnvironment = SpringBootTest.WebEnvironment.RANDOM_PORT)
@DirtiesContext(classMode = DirtiesContext.ClassMode.BEFORE_CLASS)
public class ConsulTest {

	@LocalServerPort
	private Integer schedulerPort;

	@Autowired
	EmbeddedWebApplicationContext context;

	private static ConsulUtil consulUtil;

	@BeforeClass
	public static void beforeClass() throws IOException {
		consulUtil = new ConsulUtil();
		consulUtil.start();
	}

	@AfterClass
	public static void afterClass() throws IOException, InterruptedException {

		consulUtil.stop();
	}

	@Before
	public void before() throws Exception {
	}

	@After
	public void after() {
	}

	@Test
	public void testRegisterAndDeRegisterWithConsul() {
		assertRegisterWithConsul();

		context.stop();

		assertDeRegisterWithConsul();
	}

	public void assertRegisterWithConsul() {
		ConsulClient consulClient = new ConsulClient();

		Response<Map<String, Service>> services = consulClient.getAgentServices();
		Service service = services.getValue().get("scheduler");
		assertThat(service.getService(), is("scheduler"));
		assertThat(service.getId(), is("scheduler"));
		assertThat(service.getPort(), is(schedulerPort));

		Response<Map<String, Check>> checks = consulClient.getAgentChecks();
		Check check = checks.getValue().get("service:scheduler");

		assertThat(check.getServiceName(), is("scheduler"));
		assertThat(check.getStatus(), is(Check.CheckStatus.PASSING));
		assertThat(check.getName(), is("Service 'scheduler' check"));
		assertThat(check.getCheckId(), is("service:scheduler"));
		assertThat(check.getServiceId(), is("scheduler"));
		assertThat(check.getNode(), is("0"));
	}

	private void assertDeRegisterWithConsul() {
		ConsulClient consulClient = new ConsulClient();

		Response<Map<String, Service>> services = consulClient.getAgentServices();
		Service service = services.getValue().get("scheduler");
		assertNull(service);
	}

}
