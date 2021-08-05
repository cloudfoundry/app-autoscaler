package org.cloudfoundry.autoscaler.scheduler.health;

import io.prometheus.client.hotspot.DefaultExports;

public class HealthExporter {

  private DbStatusCollector dbStatusCollector;

  public void setDbStatusCollector(DbStatusCollector dbStatusCollector) {
    this.dbStatusCollector = dbStatusCollector;
  }

  public void init() {
    DefaultExports.initialize();
    dbStatusCollector.register();
  }
}
