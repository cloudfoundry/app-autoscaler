package org.cloudfoundry.autoscaler.scheduler.health;

import java.util.ArrayList;
import java.util.List;

import javax.sql.DataSource;

import org.apache.commons.dbcp2.BasicDataSource;

import io.prometheus.client.Collector;
import io.prometheus.client.GaugeMetricFamily;

public class DBStatusCollector extends Collector {

	private String namespace = "autoscaler";
	private String subSystem = "scheduler";

	private DataSource dataSource;

	public void setDataSource(DataSource dataSource) {
		this.dataSource = dataSource;
	}

	public void setPolicyDBDataSource(DataSource policyDBDataSource) {
		this.policyDBDataSource = policyDBDataSource;
	}

	private DataSource policyDBDataSource;

	@Override
	public List<MetricFamilySamples> collect() {
		// TODO Auto-generated method stub
		List<MetricFamilySamples> mfs = new ArrayList<MetricFamilySamples>();
		BasicDataSource basicDataSource = (BasicDataSource) this.dataSource;
		BasicDataSource policyBasicDataSource = (BasicDataSource) this.policyDBDataSource;
//		primary datasource metrics
		mfs.add(new GaugeMetricFamily(namespace + "_" + subSystem + "_data_source" + "_initial_size",
				"The initial number of connections that are created when the pool is started",
				basicDataSource.getInitialSize()));
		mfs.add(new GaugeMetricFamily(namespace + "_" + subSystem + "_data_source" + "_max_active",
				"The maximum number of active connections that can be allocated from this pool at the same time, or negative for no limit",
				basicDataSource.getMaxTotal()));
		mfs.add(new GaugeMetricFamily(namespace + "_" + subSystem + "_data_source" + "_max_idle",
				"The maximum number of connections that can remain idle in the pool, without extra ones being released, or negative for no limit.",
				basicDataSource.getMaxIdle()));
		mfs.add(new GaugeMetricFamily(namespace + "_" + subSystem + "_data_source" + "_min_idle",
				"The minimum number of active connections that can remain idle in the pool, without extra ones being created, or 0 to create none.",
				basicDataSource.getMinIdle()));
		mfs.add(new GaugeMetricFamily(namespace + "_" + subSystem + "_data_source" + "_active_connections_number",
				"The current number of active connections that have been allocated from this data source",
				basicDataSource.getNumActive()));
		mfs.add(new GaugeMetricFamily(namespace + "_" + subSystem + "_data_source" + "_idle_connections_number",
				"The current number of idle connections that are waiting to be allocated from this data source",
				basicDataSource.getNumIdle()));
//		policy datasource metrics
		mfs.add(new GaugeMetricFamily(namespace + "_" + subSystem + "_policy_db_data_source" + "_active_connections_number",
				"The current number of active connections that have been allocated from this data source",
				policyBasicDataSource.getNumActive()));
		mfs.add(new GaugeMetricFamily(namespace + "_" + subSystem + "_policy_db_data_source" + "_idle_connections_number",
				"The current number of idle connections that are waiting to be allocated from this data source",
				policyBasicDataSource.getNumIdle()));
		mfs.add(new GaugeMetricFamily(namespace + "_" + subSystem + "_policy_db_data_source" + "_initial_size",
				"The initial number of connections that are created when the pool is started",
				policyBasicDataSource.getInitialSize()));
		mfs.add(new GaugeMetricFamily(namespace + "_" + subSystem + "_policy_db_data_source" + "_max_active",
				"The maximum number of active connections that can be allocated from this pool at the same time, or negative for no limit",
				policyBasicDataSource.getMaxTotal()));
		mfs.add(new GaugeMetricFamily(namespace + "_" + subSystem + "_policy_db_data_source" + "_max_idle",
				"The maximum number of connections that can remain idle in the pool, without extra ones being released, or negative for no limit.",
				policyBasicDataSource.getMaxIdle()));
		mfs.add(new GaugeMetricFamily(namespace + "_" + subSystem + "_policy_db_data_source" + "_min_idle",
				"The minimum number of active connections that can remain idle in the pool, without extra ones being created, or 0 to create none.",
				policyBasicDataSource.getMinIdle()));

		return mfs;
	}

}
