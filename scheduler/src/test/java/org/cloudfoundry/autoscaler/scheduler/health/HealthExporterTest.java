package org.cloudfoundry.autoscaler.scheduler.health;

import static org.assertj.core.api.Assertions.assertThat;

import io.prometheus.client.CollectorRegistry;
import java.util.List;
import org.cloudfoundry.autoscaler.scheduler.conf.FipsSecurityProviderConfig;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.ValueSource;

/** Fresh JVM with default providers: a requirement alone does not prove enforcement. */
class HealthExporterTest {

  @ParameterizedTest
  @ValueSource(booleans = {false, true})
  void inactiveFipsReportsBothGaugesAsZero(boolean required) {
    assertThat(FipsSecurityProviderConfig.isFipsEnabled()).isFalse();
    HealthExporter exporter = new HealthExporter(required);
    exporter.setDbStatusCollector(
        new DbStatusCollector() {
          @Override
          public List<MetricFamilySamples> collect() {
            return List.of();
          }
        });
    exporter.init();
    assertThat(CollectorRegistry.defaultRegistry.getSampleValue("autoscaler_fips_enabled"))
        .isEqualTo(0.0);
    assertThat(CollectorRegistry.defaultRegistry.getSampleValue("autoscaler_fips_enforced"))
        .isEqualTo(0.0);
  }
}
