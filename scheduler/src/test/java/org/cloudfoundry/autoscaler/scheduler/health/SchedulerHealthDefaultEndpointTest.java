package org.cloudfoundry.autoscaler.scheduler.health;

import static org.assertj.core.api.Assertions.assertThat;

import org.cloudfoundry.autoscaler.scheduler.conf.MetricsConfiguration;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.context.SpringBootTest.WebEnvironment;
import org.springframework.boot.test.web.client.TestRestTemplate;
import org.springframework.http.ResponseEntity;
import org.springframework.test.annotation.DirtiesContext;
import org.springframework.test.annotation.DirtiesContext.ClassMode;
import org.springframework.test.context.junit4.SpringRunner;

@RunWith(SpringRunner.class)
@SpringBootTest(webEnvironment = WebEnvironment.RANDOM_PORT)
@DirtiesContext(classMode = ClassMode.BEFORE_CLASS)
public class SchedulerHealthDefaultEndpointTest {

  @Autowired private TestRestTemplate restTemplate;

  @Autowired private MetricsConfiguration metricsConfig;

  @Test
  public void metricsShouldBeAvailable() {
    ResponseEntity<String> response = this.restTemplate.getForEntity(metricsUrl(), String.class);
    assertThat(response.getStatusCode().value())
        .describedAs("Http status code should be OK")
        .isEqualTo(200);
  }

  private String metricsUrl() {
    return "http://localhost:" + metricsConfig.getPort() + "/metrics";
  }
}
