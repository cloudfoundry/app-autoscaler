package org.cloudfoundry.autoscaler.scheduler.util;

import org.springframework.test.context.TestPropertySource;

@TestPropertySource(properties = { "scalingenginejob.reschedule.interval.millisecond=100",
		"scalingenginejob.reschedule.maxcount=5", "autoscaler.scalingengine.url=http://localhost:8090",
		"scalingengine.notification.reschedule.maxcount=2" })
public class TestConfiguration {
}
