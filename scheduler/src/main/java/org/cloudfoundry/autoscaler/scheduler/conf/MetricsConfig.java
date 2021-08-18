package org.cloudfoundry.autoscaler.scheduler.conf;

import io.prometheus.client.exporter.HTTPServer;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.io.IOException;

@Configuration
public class MetricsConfig {

	@Bean(destroyMethod = "stop")
	public HTTPServer metricsServer(MetricsConfiguration config) throws IOException {
		return new HTTPServer(config.getPort());
	}

}
