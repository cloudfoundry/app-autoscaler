package org.cloudfoundry.autoscaler.scheduler.health;

import io.prometheus.client.Gauge;
import io.prometheus.client.hotspot.DefaultExports;
import org.cloudfoundry.autoscaler.scheduler.conf.FipsSecurityProviderConfig;

public class HealthExporter {

  private static final Gauge FIPS_ENABLED_GAUGE =
      Gauge.build()
          .namespace("autoscaler")
          .name("fips_enabled")
          .help("Indicates whether FIPS 140-3 mode is active (1=enabled, 0=disabled).")
          .create();

  private DbStatusCollector dbStatusCollector;

  public void setDbStatusCollector(DbStatusCollector dbStatusCollector) {
    this.dbStatusCollector = dbStatusCollector;
  }

  public void init() {
    DefaultExports.initialize();
    dbStatusCollector.register();
    FIPS_ENABLED_GAUGE.set(FipsSecurityProviderConfig.isInitialized() ? 1 : 0);
    FIPS_ENABLED_GAUGE.register();
  }
}
