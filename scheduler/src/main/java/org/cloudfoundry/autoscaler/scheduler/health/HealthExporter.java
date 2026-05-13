package org.cloudfoundry.autoscaler.scheduler.health;

import io.prometheus.client.Collector;
import io.prometheus.client.CollectorRegistry;
import io.prometheus.client.hotspot.DefaultExports;
import java.util.concurrent.atomic.AtomicReference;

public class HealthExporter {

  private static final AtomicReference<Collector> registeredCollector = new AtomicReference<>();

  private DbStatusCollector dbStatusCollector;

  public void setDbStatusCollector(DbStatusCollector dbStatusCollector) {
    this.dbStatusCollector = dbStatusCollector;
  }

  public void init() {
    DefaultExports.initialize();
    Collector previous = registeredCollector.getAndSet(dbStatusCollector);
    if (previous != null) {
      CollectorRegistry.defaultRegistry.unregister(previous);
    }
    dbStatusCollector.register();
  }
}
