package org.cloudfoundry.autoscaler.scheduler.misc;

import static org.junit.Assert.assertNull;

import java.io.IOException;
import java.util.Map;

import org.cloudfoundry.autoscaler.scheduler.util.ConsulUtil;
import org.junit.AfterClass;
import org.junit.BeforeClass;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.annotation.DirtiesContext;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.context.junit4.SpringRunner;

import com.ecwid.consul.v1.ConsulClient;
import com.ecwid.consul.v1.Response;
import com.ecwid.consul.v1.agent.model.Service;

@RunWith(SpringRunner.class)
@SpringBootTest(webEnvironment = SpringBootTest.WebEnvironment.RANDOM_PORT)
@DirtiesContext(classMode = DirtiesContext.ClassMode.BEFORE_CLASS)
@TestPropertySource(locations = "classpath:application-without-consul.properties")
public class DisableConsulTest {

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

	@Test
	public void testRegisterAndDeRegisterWithConsul() {
		assertNotRegisterWithConsul();
	}

	private void assertNotRegisterWithConsul() {
		ConsulClient consulClient = new ConsulClient();

		Response<Map<String, Service>> services = consulClient.getAgentServices();
		Service service = services.getValue().get("scheduler");
		assertNull(service);
	}
}
