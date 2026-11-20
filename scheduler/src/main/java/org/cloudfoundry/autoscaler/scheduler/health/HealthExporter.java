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
          .help(
              "Indicates whether BCFIPS approved-only mode and BCJSSE FIPS mode are active at"
                  + " startup (1=enabled, 0=disabled).")
          .create();
  private static final AtomicReference<Gauge> registeredFipsGauge = new AtomicReference<>();

  private static final Gauge FIPS_ENFORCED_GAUGE =
      Gauge.build()
          .namespace("autoscaler")
          .name("fips_enforced")
          .help(
              "Indicates whether FIPS is required by configuration and verified at startup"
                  + " (1=enforced, 0=not enforced).")
          .create();
  private static final AtomicReference<Gauge> registeredFipsEnforcedGauge = new AtomicReference<>();

  private final boolean fipsRequired;

  private DbStatusCollector dbStatusCollector;

  public HealthExporter(boolean fipsRequired) {
    this.fipsRequired = fipsRequired;
  }

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

    boolean enabled = FipsSecurityProviderConfig.isFipsEnabled();
    FIPS_ENABLED_GAUGE.set(enabled ? 1 : 0);
    FIPS_ENFORCED_GAUGE.set(
        fipsRequired && FipsSecurityProviderConfig.isInitialized() && enabled ? 1 : 0);
    registerFipsGauge(FIPS_ENABLED_GAUGE, registeredFipsGauge);
    registerFipsGauge(FIPS_ENFORCED_GAUGE, registeredFipsEnforcedGauge);
  }

  private static void registerFipsGauge(Gauge gauge, AtomicReference<Gauge> registered) {
    Gauge previousFipsGauge = registered.getAndSet(gauge);
    if (previousFipsGauge != null) {
      logger.debug("Re-registering FIPS gauge after context refresh");
      CollectorRegistry.defaultRegistry.unregister(previousFipsGauge);
    }
    gauge.register();
  }
}
