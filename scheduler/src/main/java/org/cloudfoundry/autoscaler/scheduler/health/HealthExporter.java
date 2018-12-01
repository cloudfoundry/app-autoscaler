package org.cloudfoundry.autoscaler.scheduler.health;

import io.prometheus.client.hotspot.DefaultExports;

public class HealthExporter {

	private DBStatusCollector dbStatusCollector;

	public void setDbStatusCollector(DBStatusCollector dbStatusCollector) {
		this.dbStatusCollector = dbStatusCollector;
	}

	public void init() {
		DefaultExports.initialize();
		dbStatusCollector.register();
	}
}
