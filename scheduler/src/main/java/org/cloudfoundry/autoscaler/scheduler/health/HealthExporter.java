package org.cloudfoundry.autoscaler.scheduler.health;

import io.prometheus.client.Collector;
import io.prometheus.client.CollectorRegistry;
import io.prometheus.client.Gauge;
import io.prometheus.client.hotspot.DefaultExports;
import java.util.concurrent.atomic.AtomicReference;
import org.cloudfoundry.autoscaler.scheduler.conf.FipsSecurityProviderConfig;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class HealthExporter {

  private static final Logger logger = LoggerFactory.getLogger(HealthExporter.class);
  private static final AtomicReference<Collector> registeredCollector = new AtomicReference<>();

  private static final Gauge FIPS_ENABLED_GAUGE =
      Gauge.build()
          .namespace("autoscaler")
          .name("fips_enabled")
          .help("Indicates whether FIPS 140-3 mode is active (1=enabled, 0=disabled).")
          .create();
  private static final AtomicReference<Gauge> registeredFipsGauge = new AtomicReference<>();

  private DbStatusCollector dbStatusCollector;

  public void setDbStatusCollector(DbStatusCollector dbStatusCollector) {
    this.dbStatusCollector = dbStatusCollector;
  }

  public void init() {
    DefaultExports.initialize();
    Collector previous = registeredCollector.getAndSet(dbStatusCollector);
    if (previous != null) {
      logger.debug("Re-registering DbStatusCollector after context refresh");
      CollectorRegistry.defaultRegistry.unregister(previous);
    }
    dbStatusCollector.register();

    FIPS_ENABLED_GAUGE.set(FipsSecurityProviderConfig.isInitialized() ? 1 : 0);
    Gauge previousFipsGauge = registeredFipsGauge.getAndSet(FIPS_ENABLED_GAUGE);
    if (previousFipsGauge != null) {
      logger.debug("Re-registering FIPS gauge after context refresh");
      CollectorRegistry.defaultRegistry.unregister(previousFipsGauge);
    }
    FIPS_ENABLED_GAUGE.register();
  }
}
