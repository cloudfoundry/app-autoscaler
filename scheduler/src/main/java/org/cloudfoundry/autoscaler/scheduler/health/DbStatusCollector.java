package org.cloudfoundry.autoscaler.scheduler.health;

import com.zaxxer.hikari.HikariDataSource;
import io.prometheus.client.Collector;
import io.prometheus.client.GaugeMetricFamily;
import java.util.ArrayList;
import java.util.List;
import javax.sql.DataSource;

public class DbStatusCollector extends Collector {

  private String namespace = "autoscaler";
  private String subSystem = "scheduler";

  private DataSource dataSource;

  public void setDataSource(DataSource dataSource) {
    this.dataSource = dataSource;
  }

  public void setPolicyDbDataSource(DataSource policyDbDataSource) {
    this.policyDbDataSource = policyDbDataSource;
  }

  private DataSource policyDbDataSource;

  private List<MetricFamilySamples> collectForDataSource(HikariDataSource dataSource, String name) {
    List<MetricFamilySamples> mfs = new ArrayList<>();
    mfs.add(
        new GaugeMetricFamily(
            namespace + "_" + subSystem + name + "_max_size",
            "The maximum number of active connections that can be allocated from this pool at the"
                + " same time, or negative for no limit",
            dataSource.getMaximumPoolSize()));
    mfs.add(
        new GaugeMetricFamily(
            namespace + "_" + subSystem + name + "_min_idle",
            "The minimum number of active connections that can remain idle in the pool, without"
                + " extra ones being created, or 0 to create none.",
            dataSource.getMinimumIdle()));
    mfs.add(
        new GaugeMetricFamily(
            namespace + "_" + subSystem + name + "_active_connections_number",
            "The current number of active connections that have been allocated from this data"
                + " source",
            dataSource.getHikariPoolMXBean().getActiveConnections()));
    mfs.add(
        new GaugeMetricFamily(
            namespace + "_" + subSystem + name + "_idle_connections_number",
            "The current number of idle connections that are waiting to be allocated from this data"
                + " source",
            dataSource.getHikariPoolMXBean().getIdleConnections()));
    return mfs;
  }

  @Override
  public List<MetricFamilySamples> collect() {
    List<MetricFamilySamples> mfs = new ArrayList<MetricFamilySamples>();
    HikariDataSource basicDataSource = (HikariDataSource) this.dataSource;
    mfs.addAll(collectForDataSource(basicDataSource, "_data_source"));

    HikariDataSource policyBasicDataSource = (HikariDataSource) this.policyDbDataSource;
    mfs.addAll(collectForDataSource(policyBasicDataSource, "_policy_db_data_source"));
    return mfs;
  }
}
