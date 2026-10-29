package org.cloudfoundry.autoscaler.scheduler.health;

import static org.assertj.core.api.Assertions.assertThat;

import io.prometheus.client.CollectorRegistry;
import java.util.List;
import org.cloudfoundry.autoscaler.scheduler.conf.FipsSecurityProviderConfig;
import org.cloudfoundry.autoscaler.scheduler.conf.HealthConfig;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;
import org.springframework.mock.env.MockEnvironment;

/** Fresh JVM: activate FIPS before any BC classes initialize. No database is needed. */
class FipsHealthExporterTest {

  @BeforeAll
  static void enableFips() {
    FipsSecurityProviderConfig.initialize();
  }

  @Test
  void configuredAndVerifiedFipsReportsBothGauges() {
    exportMetrics(true);
    assertThat(sample("autoscaler_fips_enabled")).isEqualTo(1.0);
    assertThat(sample("autoscaler_fips_enforced")).isEqualTo(1.0);
  }

  @Test
  void runtimeFipsWithoutConfiguredRequirementIsNotEnforced() {
    exportMetrics(false);
    assertThat(sample("autoscaler_fips_enabled")).isEqualTo(1.0);
    assertThat(sample("autoscaler_fips_enforced")).isEqualTo(0.0);
  }

  @Test
  void contextRefreshRegistersBothGaugesWithoutDuplicates() {
    exportMetrics(true);
    exportMetrics(false);
    assertThat(sample("autoscaler_fips_enabled")).isEqualTo(1.0);
    assertThat(sample("autoscaler_fips_enforced")).isEqualTo(0.0);
  }

  private static void exportMetrics(boolean required) {
    DbStatusCollector collector =
        new DbStatusCollector() {
          @Override
          public List<MetricFamilySamples> collect() {
            return List.of();
          }
        };
    HealthExporter exporter =
        new HealthConfig()
            .healthExporter(
                collector,
                new MockEnvironment()
                    .withProperty("autoscaler.fips_mode", Boolean.toString(required)));
    exporter.init();
  }

  private static Double sample(String name) {
    return CollectorRegistry.defaultRegistry.getSampleValue(name);
  }
}
