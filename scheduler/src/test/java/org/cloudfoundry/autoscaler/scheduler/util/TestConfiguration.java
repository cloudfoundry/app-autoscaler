package org.cloudfoundry.autoscaler.scheduler.util;

import org.springframework.test.context.TestPropertySource;

@TestPropertySource(properties = { "scalingenginejob.reschedule.interval.millisecond=100",
		"scalingenginejob.reschedule.maxcount=5", "autoscaler.scalingengine.url=https://localhost:8091",
		"scalingengine.notification.reschedule.maxcount=2",
		"server.ssl.key-store=src/test/resources/certs/test-scheduler.p12",
		"server.ssl.key-alias=test-scheduler",
		"server.ssl.key-store-password=123456",
		"server.ssl.key-store-type=PKCS12",
		"client.ssl.key-store=src/test/resources/certs/test-scheduler.p12",
		"client.ssl.key-store-password=123456",
		"client.ssl.key-store-type=PKCS12",
		"client.ssl.trust-store=src/test/resources/certs/test.truststore",
		"client.ssl.trust-store-password=123456",
		"client.ssl.protocol=TLSv1.2"})
public class TestConfiguration {
}
