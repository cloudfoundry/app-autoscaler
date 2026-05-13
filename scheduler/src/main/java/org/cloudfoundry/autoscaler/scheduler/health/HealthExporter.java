package org.cloudfoundry.autoscaler.scheduler.health;

import io.prometheus.client.CollectorRegistry;
import io.prometheus.client.hotspot.DefaultExports;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class HealthExporter {

  private static final Logger logger = LoggerFactory.getLogger(HealthExporter.class);

  private DbStatusCollector dbStatusCollector;

  public void setDbStatusCollector(DbStatusCollector dbStatusCollector) {
    this.dbStatusCollector = dbStatusCollector;
  }

  public void init() {
    DefaultExports.initialize();
    try {
      dbStatusCollector.register();
    } catch (IllegalArgumentException e) {
      logger.info("Re-registering DbStatusCollector after context refresh");
      CollectorRegistry.defaultRegistry.clear();
      DefaultExports.initialize();
      dbStatusCollector.register();
    }
  }
}
