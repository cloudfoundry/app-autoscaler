package org.cloudfoundry.autoscaler.scheduler.rest;

import org.cloudfoundry.autoscaler.scheduler.util.ScalingEngineUtil;
import org.mockito.Mockito;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Primary;
import org.springframework.context.annotation.Profile;

@Profile("ScalingEngineUtilMock")
@Configuration
public class ScalingEngineUtilMock {
	@Bean
	@Primary
	public ScalingEngineUtil scalingEnginUtil() {
		return Mockito.mock(ScalingEngineUtil.class);
	}
}
