package org.cloudfoundry.autoscaler.scheduler.health;

import static org.assertj.core.api.Assertions.assertThat;

import org.junit.Test;
import org.junit.runner.RunWith;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.context.SpringBootTest.WebEnvironment;
import org.springframework.boot.test.web.client.TestRestTemplate;
import org.springframework.test.annotation.DirtiesContext;
import org.springframework.test.annotation.DirtiesContext.ClassMode;
import org.springframework.test.context.junit4.SpringRunner;

@RunWith(SpringRunner.class)
@SpringBootTest(webEnvironment = WebEnvironment.RANDOM_PORT)
@DirtiesContext(classMode = ClassMode.BEFORE_CLASS)
public class SchedulerHealthEndpointTest {

	@Autowired
	private TestRestTemplate restTemplate;
	@Value("${scheduler.healthserver.port}")
	private int healthServerPort;

	@Test
	public void greetingShouldReturnDefaultMessage() throws Exception {
		String result = this.restTemplate.getForObject("http://localhost:" + healthServerPort + "/metrics",
				String.class);
		assertThat(result.contains("jvm_info"));
		assertThat(result.contains("jvm_buffer_pool_used_bytes"));
		assertThat(result.contains("jvm_buffer_pool_capacity_bytes"));
		assertThat(result.contains("jvm_buffer_pool_used_buffers"));
		assertThat(result.contains("jvm_gc_collection_seconds_count"));
		assertThat(result.contains("jvm_gc_collection_seconds_sum"));
		assertThat(result.contains("jvm_classes_loaded"));
		assertThat(result.contains("jvm_classes_loaded_total"));
		assertThat(result.contains("jvm_classes_unloaded_total"));
		assertThat(result.contains("jvm_threads"));
		assertThat(result.contains("jvm_memory_bytes"));
		assertThat(result.contains("jvm_memory_pool_bytes"));
		assertThat(result.contains("autoscaler_scheduler_data_source"));
		assertThat(result.contains("autoscaler_scheduler_policy_db_data_source"));
	}

}
